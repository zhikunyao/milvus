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

package roles

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/milvus-io/milvus-proto/go-api/v2/commonpb"
	"github.com/milvus-io/milvus/cmd/components"
	"github.com/milvus-io/milvus/internal/distributed/streaming"
	"github.com/milvus-io/milvus/internal/http"
	"github.com/milvus-io/milvus/internal/http/healthz"
	"github.com/milvus-io/milvus/internal/util/dependency"
	kvfactory "github.com/milvus-io/milvus/internal/util/dependency/kv"
	"github.com/milvus-io/milvus/internal/util/initcore"
	internalmetrics "github.com/milvus-io/milvus/internal/util/metrics"
	"github.com/milvus-io/milvus/internal/util/streamingutil/util"
	"github.com/milvus-io/milvus/pkg/v2/config"
	"github.com/milvus-io/milvus/pkg/v2/log"
	"github.com/milvus-io/milvus/pkg/v2/metrics"
	rocksmqimpl "github.com/milvus-io/milvus/pkg/v2/mq/mqimpl/rocksmq/server"
	"github.com/milvus-io/milvus/pkg/v2/tracer"
	"github.com/milvus-io/milvus/pkg/v2/util/etcd"
	"github.com/milvus-io/milvus/pkg/v2/util/expr"
	"github.com/milvus-io/milvus/pkg/v2/util/gc"
	"github.com/milvus-io/milvus/pkg/v2/util/generic"
	"github.com/milvus-io/milvus/pkg/v2/util/logutil"
	"github.com/milvus-io/milvus/pkg/v2/util/metricsinfo"
	"github.com/milvus-io/milvus/pkg/v2/util/paramtable"
	_ "github.com/milvus-io/milvus/pkg/v2/util/symbolizer" // support symbolizer and crash dump
	"github.com/milvus-io/milvus/pkg/v2/util/typeutil"
)

// all milvus related metrics is in a separate registry
var Registry *internalmetrics.MilvusRegistry

func init() {
	Registry = internalmetrics.NewMilvusRegistry()
	metrics.Register(Registry.GoRegistry)
	metrics.RegisterMetaMetrics(Registry.GoRegistry)
	metrics.RegisterMsgStreamMetrics(Registry.GoRegistry)
	metrics.RegisterStorageMetrics(Registry.GoRegistry)
}

// stopRocksmqIfUsed closes the RocksMQ if it is used.
func stopRocksmqIfUsed() {
	if name := util.MustSelectWALName(); name == util.WALTypeRocksmq {
		rocksmqimpl.CloseRocksMQ()
	}
}

type component interface {
	healthz.Indicator
	Prepare() error
	Run() error
	Stop() error
}

const (
	TmpInvertedIndexPrefix = "/tmp/milvus/inverted-index/"
	TmpTextLogPrefix       = "/tmp/milvus/text-log/"
)

func cleanLocalDir(path string) {
	_, statErr := os.Stat(path)
	// path exist, but stat error
	if statErr != nil && !os.IsNotExist(statErr) {
		log.Warn("Check if path exists failed when clean local data cache", zap.Error(statErr))
		panic(statErr)
	}
	// path exist, remove all
	if statErr == nil {
		err := os.RemoveAll(path)
		if err != nil {
			log.Warn("Clean local data cache failed", zap.Error(err))
			panic(err)
		}
		log.Info("Clean local data cache", zap.String("path", path))
	}
}

func runComponent[T component](ctx context.Context,
	localMsg bool,
	runWg *sync.WaitGroup,
	creator func(context.Context, dependency.Factory) (T, error),
	metricRegister func(*prometheus.Registry),
) component {
	var role T

	sign := make(chan struct{})
	go func() {
		factory := dependency.NewFactory(localMsg)
		var err error
		role, err = creator(ctx, factory)
		if err != nil {
			panic(err)
		}
		if err := role.Prepare(); err != nil {
			panic(err)
		}
		close(sign)
		if err := role.Run(); err != nil {
			panic(err)
		}
		runWg.Done()
	}()

	<-sign

	healthz.Register(role)
	metricRegister(Registry.GoRegistry)
	if generic.IsZero(role) {
		return nil
	}
	return role
}

// MilvusRoles decides which components are brought up with Milvus.
type MilvusRoles struct {
	EnableProxy         bool `env:"ENABLE_PROXY"`
	EnableMixCoord      bool `env:"ENABLE_ROOT_COORD"`
	EnableQueryNode     bool `env:"ENABLE_QUERY_NODE"`
	EnableDataNode      bool `env:"ENABLE_DATA_NODE"`
	EnableStreamingNode bool `env:"ENABLE_STREAMING_NODE"`
	EnableRootCoord     bool `env:"ENABLE_ROOT_COORD"`
	EnableQueryCoord    bool `env:"ENABLE_QUERY_COORD"`
	EnableDataCoord     bool `env:"ENABLE_DATA_COORD"`
	Local               bool
	Alias               string
	Embedded            bool

	ServerType string

	closed chan struct{}
	once   sync.Once
}

// NewMilvusRoles creates a new MilvusRoles with private fields initialized.
func NewMilvusRoles() *MilvusRoles {
	mr := &MilvusRoles{
		closed: make(chan struct{}),
	}
	return mr
}

func (mr *MilvusRoles) printLDPreLoad() {
	const LDPreLoad = "LD_PRELOAD"
	val, ok := os.LookupEnv(LDPreLoad)
	if ok {
		log.Info("Enable Jemalloc", zap.String("Jemalloc Path", val))
	}
}

func (mr *MilvusRoles) runProxy(ctx context.Context, localMsg bool, wg *sync.WaitGroup) component {
	wg.Add(1)
	return runComponent(ctx, localMsg, wg, components.NewProxy, metrics.RegisterProxy)
}

func (mr *MilvusRoles) runMixCoord(ctx context.Context, localMsg bool, wg *sync.WaitGroup) component {
	wg.Add(1)
	return runComponent(ctx, localMsg, wg, components.NewMixCoord, metrics.RegisterMixCoord)
}

func (mr *MilvusRoles) runQueryNode(ctx context.Context, localMsg bool, wg *sync.WaitGroup) component {
	wg.Add(1)
	// clear local storage
	rootPath := paramtable.Get().LocalStorageCfg.Path.GetValue()
	queryDataLocalPath := filepath.Join(rootPath, typeutil.QueryNodeRole)
	cleanLocalDir(queryDataLocalPath)
	// clear mmap dir
	mmapDir := paramtable.Get().QueryNodeCfg.MmapDirPath.GetValue()
	if len(mmapDir) > 0 {
		cleanLocalDir(mmapDir)
	}
	cleanLocalDir(TmpInvertedIndexPrefix)
	cleanLocalDir(TmpTextLogPrefix)

	return runComponent(ctx, localMsg, wg, components.NewQueryNode, metrics.RegisterQueryNode)
}

func (mr *MilvusRoles) runStreamingNode(ctx context.Context, localMsg bool, wg *sync.WaitGroup) component {
	wg.Add(1)
	return runComponent(ctx, localMsg, wg, components.NewStreamingNode, metrics.RegisterStreamingNode)
}

func (mr *MilvusRoles) runDataNode(ctx context.Context, localMsg bool, wg *sync.WaitGroup) component {
	wg.Add(1)
	return runComponent(ctx, localMsg, wg, components.NewDataNode, metrics.RegisterDataNode)
}

func (mr *MilvusRoles) setupLogger() {
	params := paramtable.Get()
	logConfig := log.Config{
		Level:     params.LogCfg.Level.GetValue(),
		GrpcLevel: params.LogCfg.GrpcLogLevel.GetValue(),
		Format:    params.LogCfg.Format.GetValue(),
		Stdout:    params.LogCfg.Stdout.GetAsBool(),
		File: log.FileLogConfig{
			RootPath:   params.LogCfg.RootPath.GetValue(),
			MaxSize:    params.LogCfg.MaxSize.GetAsInt(),
			MaxDays:    params.LogCfg.MaxAge.GetAsInt(),
			MaxBackups: params.LogCfg.MaxBackups.GetAsInt(),
		},
	}
	id := paramtable.GetNodeID()
	roleName := paramtable.GetRole()
	rootPath := logConfig.File.RootPath
	if rootPath != "" {
		logConfig.File.Filename = fmt.Sprintf("%s-%d.log", roleName, id)
	} else {
		logConfig.File.Filename = ""
	}

	logutil.SetupLogger(&logConfig)
	params.Watch(params.LogCfg.Level.Key, config.NewHandler("log.level", func(event *config.Event) {
		if !event.HasUpdated || event.EventType == config.DeleteType {
			return
		}
		logLevel, err := zapcore.ParseLevel(event.Value)
		if err != nil {
			log.Warn("failed to parse log level", zap.Error(err))
			return
		}
		log.SetLevel(logLevel)
		log.Info("log level changed", zap.String("level", event.Value))
	}))
}

// Register serves prometheus http service
func setupPrometheusHTTPServer(r *internalmetrics.MilvusRegistry) {
	log.Info("setupPrometheusHTTPServer")
	http.Register(&http.Handler{
		Path:    http.MetricsPath,
		Handler: promhttp.HandlerFor(r, promhttp.HandlerOpts{}),
	})
	http.Register(&http.Handler{
		Path:    http.MetricsDefaultPath,
		Handler: promhttp.Handler(),
	})
}

func (mr *MilvusRoles) handleSignals() func() {
	sign := make(chan struct{})
	done := make(chan struct{})

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		defer close(done)
		for {
			select {
			case <-sign:
				log.Info("All cleanup done, handleSignals goroutine quit")
				return
			case sig := <-sc:
				log.Warn("Get signal to exit", zap.String("signal", sig.String()))
				mr.once.Do(func() {
					close(mr.closed)
					// reset other signals, only handle SIGINT from now
					signal.Reset(syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGTERM)
				})
			}
		}
	}()
	return func() {
		close(sign)
		<-done
	}
}

// Run Milvus components.
func (mr *MilvusRoles) Run() {
	// start signal handler, defer close func
	closeFn := mr.handleSignals()
	defer closeFn()

	log.Info("starting running Milvus components")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mr.printLDPreLoad()

	// start milvus thread watcher to update actual thread number metrics
	thw := internalmetrics.NewThreadWatcher()
	thw.Start()
	defer thw.Stop()

	internalmetrics.InitHolmes()
	defer internalmetrics.CloseHolmes()

	// only standalone enable localMsg
	if mr.Local {
		if err := os.Setenv(metricsinfo.DeployModeEnvKey, metricsinfo.StandaloneDeployMode); err != nil {
			log.Error("Failed to set deploy mode: ", zap.Error(err))
		}

		if mr.Embedded {
			// setup config for embedded milvus
			paramtable.InitWithBaseTable(paramtable.NewBaseTable(paramtable.Files([]string{"embedded-milvus.yaml"})))
		} else {
			paramtable.Init()
		}

		params := paramtable.Get()
		if params.EtcdCfg.UseEmbedEtcd.GetAsBool() {
			// Start etcd server.
			etcd.InitEtcdServer(
				params.EtcdCfg.UseEmbedEtcd.GetAsBool(),
				params.EtcdCfg.ConfigPath.GetValue(),
				params.EtcdCfg.DataDir.GetValue(),
				params.EtcdCfg.EtcdLogPath.GetValue(),
				params.EtcdCfg.EtcdLogLevel.GetValue())
			defer etcd.StopEtcdServer()
		}
		paramtable.SetRole(typeutil.StandaloneRole)
		defer stopRocksmqIfUsed()
	} else {
		if err := os.Setenv(metricsinfo.DeployModeEnvKey, metricsinfo.ClusterDeployMode); err != nil {
			log.Error("Failed to set deploy mode: ", zap.Error(err))
		}
		paramtable.Init()
		paramtable.SetRole(mr.ServerType)
	}

	// init tracer before run any component
	tracer.Init()

	// Initialize streaming service if enabled.

	if mr.ServerType == typeutil.StandaloneRole || !mr.EnableDataNode {
		// only datanode does not init streaming service
		streaming.Init()
		defer streaming.Release()
	}

	enableComponents := []bool{
		mr.EnableProxy,
		mr.EnableQueryNode,
		mr.EnableDataNode,
		mr.EnableStreamingNode,
		mr.EnableMixCoord,
		mr.EnableRootCoord,
		mr.EnableQueryCoord,
		mr.EnableDataCoord,
	}
	enableComponents = lo.Filter(enableComponents, func(v bool, _ int) bool {
		return v
	})
	healthz.SetComponentNum(len(enableComponents))

	expr.Init()
	expr.Register("param", paramtable.Get())
	mr.setupLogger()
	http.ServeHTTP()
	setupPrometheusHTTPServer(Registry)

	if paramtable.Get().CommonCfg.GCEnabled.GetAsBool() {
		if paramtable.Get().CommonCfg.GCHelperEnabled.GetAsBool() {
			action := func(GOGC uint32) {
				debug.SetGCPercent(int(GOGC))
			}
			gc.NewTuner(paramtable.Get().CommonCfg.OverloadedMemoryThresholdPercentage.GetAsFloat(), uint32(paramtable.Get().CommonCfg.MinimumGOGCConfig.GetAsInt()), uint32(paramtable.Get().CommonCfg.MaximumGOGCConfig.GetAsInt()), action)
		} else {
			action := func(uint32) {}
			gc.NewTuner(paramtable.Get().CommonCfg.OverloadedMemoryThresholdPercentage.GetAsFloat(), uint32(paramtable.Get().CommonCfg.MinimumGOGCConfig.GetAsInt()), uint32(paramtable.Get().CommonCfg.MaximumGOGCConfig.GetAsInt()), action)
		}
	}

	var wg sync.WaitGroup
	local := mr.Local

	componentMap := make(map[string]component)
	var mixCoord component
	var proxy, dataNode, queryNode, streamingNode component

	if (mr.EnableRootCoord && mr.EnableDataCoord && mr.EnableQueryCoord) || mr.EnableMixCoord {
		paramtable.SetLocalComponentEnabled(typeutil.MixCoordRole)
		mixCoord = mr.runMixCoord(ctx, local, &wg)
		componentMap[typeutil.MixCoordRole] = mixCoord
	}

	if mr.EnableQueryNode {
		paramtable.SetLocalComponentEnabled(typeutil.QueryNodeRole)
		queryNode = mr.runQueryNode(ctx, local, &wg)
		componentMap[typeutil.QueryNodeRole] = queryNode
	}

	if mr.EnableDataNode {
		paramtable.SetLocalComponentEnabled(typeutil.DataNodeRole)
		dataNode = mr.runDataNode(ctx, local, &wg)
		componentMap[typeutil.DataNodeRole] = dataNode
	}

	if mr.EnableProxy {
		paramtable.SetLocalComponentEnabled(typeutil.ProxyRole)
		proxy = mr.runProxy(ctx, local, &wg)
		componentMap[typeutil.ProxyRole] = proxy
	}

	if mr.EnableStreamingNode {
		// Before initializing the local streaming node, make sure the local registry is ready.
		paramtable.SetLocalComponentEnabled(typeutil.StreamingNodeRole)
		streamingNode = mr.runStreamingNode(ctx, local, &wg)
		componentMap[typeutil.StreamingNodeRole] = streamingNode
	}

	wg.Wait()

	http.RegisterStopComponent(func(role string) error {
		if len(role) == 0 || componentMap[role] == nil {
			return fmt.Errorf("stop component [%s] in [%s] is not supported", role, mr.ServerType)
		}

		log.Info("unregister component before stop", zap.String("role", role))
		healthz.UnRegister(role)
		return componentMap[role].Stop()
	})

	http.RegisterCheckComponentReady(func(role string) error {
		if len(role) == 0 || componentMap[role] == nil {
			return fmt.Errorf("check component state for [%s] in [%s] is not supported", role, mr.ServerType)
		}

		// for coord component, if it's in standby state, it will return StateCode_StandBy
		code := componentMap[role].Health(context.TODO())
		if code != commonpb.StateCode_Healthy {
			return fmt.Errorf("component [%s] in [%s] is not healthy", role, mr.ServerType)
		}

		return nil
	})

	paramtable.Get().WatchKeyPrefix("trace", config.NewHandler("tracing handler", func(e *config.Event) {
		params := paramtable.Get()

		exp, err := tracer.CreateTracerExporter(params)
		if err != nil {
			log.Warn("Init tracer faield", zap.Error(err))
			return
		}

		// close old provider
		err = tracer.CloseTracerProvider(context.Background())
		if err != nil {
			log.Warn("Close old provider failed, stop reset", zap.Error(err))
			return
		}

		tracer.SetTracerProvider(exp, params.TraceCfg.SampleFraction.GetAsFloat())
		log.Info("Reset tracer finished", zap.String("Exporter", params.TraceCfg.Exporter.GetValue()), zap.Float64("SampleFraction", params.TraceCfg.SampleFraction.GetAsFloat()))

		tracer.NotifyTracerProviderUpdated()

		if paramtable.GetRole() == typeutil.QueryNodeRole || paramtable.GetRole() == typeutil.StandaloneRole {
			initcore.ResetTraceConfig(params)
			log.Info("Reset segcore tracer finished", zap.String("Exporter", params.TraceCfg.Exporter.GetValue()))
		}
	}))

	paramtable.SetCreateTime(time.Now())
	paramtable.SetUpdateTime(time.Now())

	<-mr.closed

	// stop coordinators first
	coordinators := []component{mixCoord}
	for idx, coord := range coordinators {
		log.Warn("stop processing")
		if coord != nil {
			log.Info("stop coord", zap.Int("idx", idx), zap.Any("coord", coord))
			coord.Stop()
		}
	}
	log.Info("All coordinators have stopped")

	// stop nodes
	nodes := []component{streamingNode, queryNode, dataNode}
	stopNodeWG := &sync.WaitGroup{}
	for _, node := range nodes {
		if node != nil {
			stopNodeWG.Add(1)
			go func() {
				defer func() {
					stopNodeWG.Done()
					log.Info("stop node done", zap.Any("node", node))
				}()
				log.Info("stop node...", zap.Any("node", node))
				node.Stop()
			}()
		}
	}
	stopNodeWG.Wait()
	log.Info("All nodes have stopped")

	if proxy != nil {
		proxy.Stop()
		log.Info("proxy stopped!")
	}

	// close reused etcd client
	kvfactory.CloseEtcdClient()

	log.Info("Milvus components graceful stop done")
}

func (mr *MilvusRoles) GetRoles() []string {
	roles := make([]string, 0)
	if mr.EnableMixCoord {
		roles = append(roles, typeutil.MixCoordRole)
	}
	if mr.EnableProxy {
		roles = append(roles, typeutil.ProxyRole)
	}
	if mr.EnableQueryNode {
		roles = append(roles, typeutil.QueryNodeRole)
	}
	if mr.EnableDataNode {
		roles = append(roles, typeutil.DataNodeRole)
	}
	return roles
}
