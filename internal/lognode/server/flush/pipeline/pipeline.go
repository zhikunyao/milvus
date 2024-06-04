// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipeline

import (
	"context"
	"path"
	"sync"
	"time"

	"github.com/milvus-io/milvus/internal/lognode/server"
	"github.com/milvus-io/milvus/internal/lognode/server/flush/io"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/milvus-io/milvus-proto/go-api/v2/schemapb"
	"github.com/milvus-io/milvus/internal/datanode/allocator"
	"github.com/milvus-io/milvus/internal/datanode/broker"
	"github.com/milvus-io/milvus/internal/datanode/metacache"
	"github.com/milvus-io/milvus/internal/datanode/syncmgr"
	"github.com/milvus-io/milvus/internal/datanode/writebuffer"
	"github.com/milvus-io/milvus/internal/proto/datapb"
	"github.com/milvus-io/milvus/internal/querycoordv2/params"
	"github.com/milvus-io/milvus/internal/storage"
	"github.com/milvus-io/milvus/internal/util/flowgraph"
	"github.com/milvus-io/milvus/pkg/log"
	"github.com/milvus-io/milvus/pkg/metrics"
	"github.com/milvus-io/milvus/pkg/mq/msgdispatcher"
	"github.com/milvus-io/milvus/pkg/mq/msgstream"
	"github.com/milvus-io/milvus/pkg/util/conc"
	"github.com/milvus-io/milvus/pkg/util/funcutil"
	"github.com/milvus-io/milvus/pkg/util/paramtable"
	"github.com/milvus-io/milvus/pkg/util/typeutil"
)

// Pipeline controls a flowgraph for a specific collection
type Pipeline struct {
	ctx          context.Context
	cancelFn     context.CancelFunc
	metacache    metacache.MetaCache
	collectionID UniqueID // collection id of vchan for which this data sync service serves
	vchannelName string

	// TODO: should be equal to paramtable.GetNodeID(), but intergrationtest has 1 paramtable for a minicluster, the NodeID
	// varies, will cause savebinglogpath check fail. So we pass ServerID into Pipeline to aviod it failure.
	serverID UniqueID

	fg *flowgraph.TimeTickedFlowGraph // internal flowgraph processes insert/delta messages

	broker  broker.Broker
	syncMgr syncmgr.SyncManager

	timetickSender *timeTickSender // reference to timeTickSender
	dispClient     msgdispatcher.Client
	chunkManager   storage.ChunkManager

	stopOnce sync.Once
}

type nodeConfig struct {
	msFactory    msgstream.Factory // msgStream factory
	collectionID UniqueID
	vChannelName string
	metacache    metacache.MetaCache
	allocator    allocator.Allocator
	serverID     UniqueID
}

// start the flow graph in Pipeline
func (p *Pipeline) start() {
	if p.fg != nil {
		log.Info("Pipeline starting flow graph", zap.Int64("collectionID", p.collectionID),
			zap.String("vChanName", p.vchannelName))
		p.fg.Start()
	} else {
		log.Warn("Pipeline starting flow graph is nil", zap.Int64("collectionID", p.collectionID),
			zap.String("vChanName", p.vchannelName))
	}
}

func (p *Pipeline) GracefullyClose() {
	if p.fg != nil {
		log.Info("Pipeline gracefully closing flowgraph")
		p.fg.SetCloseMethod(flowgraph.CloseGracefully)
		p.close()
	}
}

func (p *Pipeline) GetMetaCache() metacache.MetaCache {
	return p.metacache
}

func (p *Pipeline) close() {
	p.stopOnce.Do(func() {
		log := log.Ctx(p.ctx).With(
			zap.Int64("collectionID", p.collectionID),
			zap.String("vChanName", p.vchannelName),
		)
		if p.fg != nil {
			log.Info("Pipeline closing flowgraph")
			p.dispClient.Deregister(p.vchannelName)
			p.fg.Close()
			log.Info("Pipeline flowgraph closed")
		}

		p.cancelFn()

		// clean up metrics
		pChan := funcutil.ToPhysicalChannel(p.vchannelName)
		metrics.CleanupDataNodeCollectionMetrics(paramtable.GetNodeID(), p.collectionID, pChan)

		log.Info("Pipeline closed")
	})
}

func initMetaCache(initCtx context.Context, storageV2Cache *metacache.StorageV2Cache, chunkManager storage.ChunkManager, info *datapb.ChannelWatchInfo, unflushed, flushed []*datapb.SegmentInfo) (metacache.MetaCache, error) {
	// tickler will update addSegment progress to watchInfo
	futures := make([]*conc.Future[any], 0, len(unflushed)+len(flushed))
	segmentPks := typeutil.NewConcurrentMap[int64, []*storage.PkStatistics]()

	loadSegmentStats := func(segType string, segments []*datapb.SegmentInfo) {
		for _, item := range segments {
			log.Info("recover segments from checkpoints",
				zap.String("vChannelName", item.GetInsertChannel()),
				zap.Int64("segmentID", item.GetID()),
				zap.Int64("numRows", item.GetNumOfRows()),
				zap.String("segmentType", segType),
			)
			segment := item

			future := io.GetOrCreateIOPool().Submit(func() (any, error) {
				var stats []*storage.PkStatistics
				var err error
				if params.Params.CommonCfg.EnableStorageV2.GetAsBool() {
					stats, err = loadStatsV2(storageV2Cache, segment, info.GetSchema())
				} else {
					stats, err = loadStats(initCtx, chunkManager, info.GetSchema(), segment.GetID(), segment.GetStatslogs())
				}
				if err != nil {
					return nil, err
				}
				segmentPks.Insert(segment.GetID(), stats)

				return struct{}{}, nil
			})

			futures = append(futures, future)
		}
	}

	loadSegmentStats("growing", unflushed)
	loadSegmentStats("sealed", flushed)

	// use fetched segment info
	info.Vchan.FlushedSegments = flushed
	info.Vchan.UnflushedSegments = unflushed

	if err := conc.AwaitAll(futures...); err != nil {
		return nil, err
	}

	// return channel, nil
	metacache := metacache.NewMetaCache(info, func(segment *datapb.SegmentInfo) *metacache.BloomFilterSet {
		entries, _ := segmentPks.Get(segment.GetID())
		return metacache.NewBloomFilterSet(entries...)
	})

	return metacache, nil
}

func loadStatsV2(storageCache *metacache.StorageV2Cache, segment *datapb.SegmentInfo, schema *schemapb.CollectionSchema) ([]*storage.PkStatistics, error) {
	space, err := storageCache.GetOrCreateSpace(segment.ID, syncmgr.SpaceCreatorFunc(segment.ID, schema, storageCache.ArrowSchema()))
	if err != nil {
		return nil, err
	}

	getResult := func(stats []*storage.PrimaryKeyStats) []*storage.PkStatistics {
		result := make([]*storage.PkStatistics, 0, len(stats))
		for _, stat := range stats {
			pkStat := &storage.PkStatistics{
				PkFilter: stat.BF,
				MinPK:    stat.MinPk,
				MaxPK:    stat.MaxPk,
			}
			result = append(result, pkStat)
		}
		return result
	}

	blobs := space.StatisticsBlobs()
	deserBlobs := make([]*Blob, 0)
	for _, b := range blobs {
		if b.Name == storage.CompoundStatsType.LogIdx() {
			blobData := make([]byte, b.Size)
			_, err = space.ReadBlob(b.Name, blobData)
			if err != nil {
				return nil, err
			}
			stats, err := storage.DeserializeStatsList(&Blob{Value: blobData})
			if err != nil {
				return nil, err
			}
			return getResult(stats), nil
		}
	}

	for _, b := range blobs {
		blobData := make([]byte, b.Size)
		_, err = space.ReadBlob(b.Name, blobData)
		if err != nil {
			return nil, err
		}
		deserBlobs = append(deserBlobs, &Blob{Value: blobData})
	}
	stats, err := storage.DeserializeStats(deserBlobs)
	if err != nil {
		return nil, err
	}
	return getResult(stats), nil
}

func loadStats(ctx context.Context, chunkManager storage.ChunkManager, schema *schemapb.CollectionSchema, segmentID int64, statsBinlogs []*datapb.FieldBinlog) ([]*storage.PkStatistics, error) {
	_, span := otel.Tracer(typeutil.DataNodeRole).Start(ctx, "loadStats")
	defer span.End()

	startTs := time.Now()
	log := log.Ctx(ctx).With(zap.Int64("segmentID", segmentID))
	log.Info("begin to init pk bloom filter", zap.Int("statsBinLogsLen", len(statsBinlogs)))

	pkField, err := typeutil.GetPrimaryFieldSchema(schema)
	if err != nil {
		return nil, err
	}

	// filter stats binlog files which is pk field stats log
	bloomFilterFiles := []string{}
	logType := storage.DefaultStatsType

	for _, binlog := range statsBinlogs {
		if binlog.FieldID != pkField.GetFieldID() {
			continue
		}
	Loop:
		for _, log := range binlog.GetBinlogs() {
			_, logidx := path.Split(log.GetLogPath())
			// if special status log exist
			// only load one file
			switch logidx {
			case storage.CompoundStatsType.LogIdx():
				bloomFilterFiles = []string{log.GetLogPath()}
				logType = storage.CompoundStatsType
				break Loop
			default:
				bloomFilterFiles = append(bloomFilterFiles, log.GetLogPath())
			}
		}
	}

	// no stats log to parse, initialize a new BF
	if len(bloomFilterFiles) == 0 {
		log.Warn("no stats files to load")
		return nil, nil
	}

	// read historical PK filter
	values, err := chunkManager.MultiRead(ctx, bloomFilterFiles)
	if err != nil {
		log.Warn("failed to load bloom filter files", zap.Error(err))
		return nil, err
	}
	blobs := make([]*Blob, 0)
	for i := 0; i < len(values); i++ {
		blobs = append(blobs, &Blob{Value: values[i]})
	}

	var stats []*storage.PrimaryKeyStats
	if logType == storage.CompoundStatsType {
		stats, err = storage.DeserializeStatsList(blobs[0])
		if err != nil {
			log.Warn("failed to deserialize stats list", zap.Error(err))
			return nil, err
		}
	} else {
		stats, err = storage.DeserializeStats(blobs)
		if err != nil {
			log.Warn("failed to deserialize stats", zap.Error(err))
			return nil, err
		}
	}

	var size uint
	result := make([]*storage.PkStatistics, 0, len(stats))
	for _, stat := range stats {
		pkStat := &storage.PkStatistics{
			PkFilter: stat.BF,
			MinPK:    stat.MinPk,
			MaxPK:    stat.MaxPk,
		}
		size += stat.BF.Cap()
		result = append(result, pkStat)
	}

	log.Info("Successfully load pk stats", zap.Any("time", time.Since(startTs)), zap.Uint("size", size))
	return result, nil
}

func getServiceWithChannel(initCtx context.Context, node *server.DataNode, info *datapb.ChannelWatchInfo, metacache metacache.MetaCache, storageV2Cache *metacache.StorageV2Cache, unflushed, flushed []*datapb.SegmentInfo) (*Pipeline, error) {
	var (
		channelName  = info.GetVchan().GetChannelName()
		collectionID = info.GetVchan().GetCollectionID()
	)

	config := &nodeConfig{
		msFactory: node.factory,
		allocator: node.allocator,

		collectionID: collectionID,
		vChannelName: channelName,
		metacache:    metacache,
		serverID:     node.session.ServerID,
	}

	err := node.writeBufferManager.Register(channelName, metacache, storageV2Cache, writebuffer.WithMetaWriter(syncmgr.BrokerMetaWriter(node.broker, config.serverID)), writebuffer.WithIDAllocator(node.allocator))
	if err != nil {
		log.Warn("failed to register channel buffer", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err != nil {
			defer node.writeBufferManager.RemoveChannel(channelName)
		}
	}()

	ctx, cancel := context.WithCancel(node.ctx)
	ds := &Pipeline{
		ctx:      ctx,
		cancelFn: cancel,

		dispClient: node.dispClient,
		msFactory:  node.factory,
		broker:     node.broker,

		metacache:    config.metacache,
		collectionID: config.collectionID,
		vchannelName: config.vChannelName,
		serverID:     config.serverID,

		chunkManager:   node.chunkManager,
		timetickSender: node.timeTickSender,
		syncMgr:        node.syncMgr,

		fg: nil,
	}

	// init flowgraph
	fg := flowgraph.NewTimeTickedFlowGraph(node.ctx)
	dmStreamNode, err := newDmInputNode(initCtx, node.dispClient, info.GetVchan().GetSeekPosition(), config)
	if err != nil {
		return nil, err
	}

	ddNode, err := newDDNode(
		node.ctx,
		collectionID,
		channelName,
		info.GetVchan().GetDroppedSegmentIds(),
		flushed,
		unflushed,
	)
	if err != nil {
		return nil, err
	}

	updater := ds.timetickSender
	writeNode := newWriteNode(node.ctx, node.writeBufferManager, updater, config)

	ttNode, err := newTTNode(config, node.writeBufferManager, node.channelCheckpointUpdater)
	if err != nil {
		return nil, err
	}

	if err := fg.AssembleNodes(dmStreamNode, ddNode, writeNode, ttNode); err != nil {
		return nil, err
	}
	ds.fg = fg

	return ds, nil
}

// newDataSyncService gets a Pipeline, but pipelines are not running
// initCtx is used to init the Pipeline only, if initCtx.Canceled or initCtx.Timeout
// newDataSyncService stops and returns the initCtx.Err()
// NOTE: compactiable for event manager
func newDataSyncService(initCtx context.Context, node *server.DataNode, info *datapb.ChannelWatchInfo) (*Pipeline, error) {
	// recover segment checkpoints
	unflushedSegmentInfos, err := node.broker.GetSegmentInfo(initCtx, info.GetVchan().GetUnflushedSegmentIds())
	if err != nil {
		return nil, err
	}
	flushedSegmentInfos, err := node.broker.GetSegmentInfo(initCtx, info.GetVchan().GetFlushedSegmentIds())
	if err != nil {
		return nil, err
	}

	var storageCache *metacache.StorageV2Cache
	if params.Params.CommonCfg.EnableStorageV2.GetAsBool() {
		storageCache, err = metacache.NewStorageV2Cache(info.Schema)
		if err != nil {
			return nil, err
		}
	}
	// init metaCache meta
	metaCache, err := initMetaCache(initCtx, storageCache, node.chunkManager, info, unflushedSegmentInfos, flushedSegmentInfos)
	if err != nil {
		return nil, err
	}

	return getServiceWithChannel(initCtx, node, info, metaCache, storageCache, unflushedSegmentInfos, flushedSegmentInfos)
}