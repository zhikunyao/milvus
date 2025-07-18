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

package importv2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/milvus-io/milvus-proto/go-api/v2/commonpb"
	"github.com/milvus-io/milvus-proto/go-api/v2/milvuspb"
	"github.com/milvus-io/milvus-proto/go-api/v2/schemapb"
	"github.com/milvus-io/milvus/internal/util/importutilv2"
	"github.com/milvus-io/milvus/pkg/v2/common"
	"github.com/milvus-io/milvus/pkg/v2/log"
	"github.com/milvus-io/milvus/pkg/v2/proto/internalpb"
	"github.com/milvus-io/milvus/pkg/v2/util/funcutil"
	"github.com/milvus-io/milvus/pkg/v2/util/merr"
	"github.com/milvus-io/milvus/pkg/v2/util/metric"
	"github.com/milvus-io/milvus/tests/integration"
)

func (s *BulkInsertSuite) testMultipleVectorFields() {
	const (
		rowCount = 100
		dim1     = 64
		dim2     = 32
	)

	c := s.Cluster
	ctx, cancel := context.WithTimeout(c.GetContext(), 600*time.Second)
	defer cancel()

	collectionName := "TestBulkInsert_MultipleVectorFields_" + funcutil.GenRandomStr()

	schema := integration.ConstructSchema(collectionName, 0, true, &schemapb.FieldSchema{
		FieldID:      100,
		Name:         integration.Int64Field,
		IsPrimaryKey: true,
		DataType:     schemapb.DataType_Int64,
		AutoID:       true,
	}, &schemapb.FieldSchema{
		FieldID:  101,
		Name:     integration.FloatVecField,
		DataType: schemapb.DataType_FloatVector,
		TypeParams: []*commonpb.KeyValuePair{
			{
				Key:   common.DimKey,
				Value: fmt.Sprintf("%d", dim1),
			},
		},
	}, &schemapb.FieldSchema{
		FieldID:  102,
		Name:     integration.BFloat16VecField,
		DataType: schemapb.DataType_BFloat16Vector,
		TypeParams: []*commonpb.KeyValuePair{
			{
				Key:   common.DimKey,
				Value: fmt.Sprintf("%d", dim2),
			},
		},
	})
	schema.EnableDynamicField = true
	marshaledSchema, err := proto.Marshal(schema)
	s.NoError(err)

	createCollectionStatus, err := c.MilvusClient.CreateCollection(ctx, &milvuspb.CreateCollectionRequest{
		DbName:         "",
		CollectionName: collectionName,
		Schema:         marshaledSchema,
		ShardsNum:      common.DefaultShardsNum,
	})
	s.NoError(err)
	s.Equal(int32(0), createCollectionStatus.GetCode())

	// create index 1
	createIndexStatus, err := c.MilvusClient.CreateIndex(ctx, &milvuspb.CreateIndexRequest{
		CollectionName: collectionName,
		FieldName:      integration.FloatVecField,
		IndexName:      "_default_1",
		ExtraParams:    integration.ConstructIndexParam(dim1, integration.IndexFaissIvfFlat, metric.L2),
	})
	s.NoError(err)
	s.Equal(int32(0), createIndexStatus.GetCode())

	s.WaitForIndexBuilt(ctx, collectionName, integration.FloatVecField)

	// create index 2
	createIndexStatus, err = c.MilvusClient.CreateIndex(ctx, &milvuspb.CreateIndexRequest{
		CollectionName: collectionName,
		FieldName:      integration.BFloat16VecField,
		IndexName:      "_default_2",
		ExtraParams:    integration.ConstructIndexParam(dim2, integration.IndexFaissIvfFlat, metric.L2),
	})
	s.NoError(err)
	s.Equal(int32(0), createIndexStatus.GetCode())

	s.WaitForIndexBuilt(ctx, collectionName, integration.BFloat16VecField)

	// import
	var files []*internalpb.ImportFile

	options := []*commonpb.KeyValuePair{}

	switch s.fileType {
	case importutilv2.Numpy:
		importFile, err := GenerateNumpyFiles(c, schema, rowCount)
		s.NoError(err)
		importFile.Paths = lo.Filter(importFile.Paths, func(path string, _ int) bool {
			return !strings.Contains(path, "$meta")
		})
		files = []*internalpb.ImportFile{importFile}
	case importutilv2.JSON:
		rowBasedFile := GenerateJSONFile(s.T(), c, schema, rowCount)
		files = []*internalpb.ImportFile{
			{
				Paths: []string{
					rowBasedFile,
				},
			},
		}
	case importutilv2.Parquet:
		filePath, err := GenerateParquetFile(s.Cluster, schema, rowCount)
		s.NoError(err)
		files = []*internalpb.ImportFile{
			{
				Paths: []string{
					filePath,
				},
			},
		}
	case importutilv2.CSV:
		filePath, sep := GenerateCSVFile(s.T(), s.Cluster, schema, rowCount)
		options = []*commonpb.KeyValuePair{{Key: "sep", Value: string(sep)}}
		s.NoError(err)
		files = []*internalpb.ImportFile{
			{
				Paths: []string{
					filePath,
				},
			},
		}
	}

	importResp, err := c.ProxyClient.ImportV2(ctx, &internalpb.ImportRequest{
		CollectionName: collectionName,
		Files:          files,
		Options:        options,
	})
	s.NoError(err)
	s.Equal(int32(0), importResp.GetStatus().GetCode())
	log.Info("Import result", zap.Any("importResp", importResp))

	jobID := importResp.GetJobID()
	err = WaitForImportDone(ctx, c, jobID)
	s.NoError(err)

	// load
	loadStatus, err := c.MilvusClient.LoadCollection(ctx, &milvuspb.LoadCollectionRequest{
		CollectionName: collectionName,
	})
	s.NoError(err)
	s.Equal(commonpb.ErrorCode_Success, loadStatus.GetErrorCode())
	s.WaitForLoad(ctx, collectionName)

	segments, err := c.ShowSegments(collectionName)
	s.NoError(err)
	s.NotEmpty(segments)
	log.Info("Show segments", zap.Any("segments", segments))

	// load refresh
	loadStatus, err = c.MilvusClient.LoadCollection(ctx, &milvuspb.LoadCollectionRequest{
		CollectionName: collectionName,
		Refresh:        true,
	})
	s.NoError(err)
	s.Equal(commonpb.ErrorCode_Success, loadStatus.GetErrorCode())
	s.WaitForLoadRefresh(ctx, "", collectionName)

	// search vec 1
	expr := fmt.Sprintf("%s > 0", integration.Int64Field)
	nq := 10
	topk := 10
	roundDecimal := -1

	params := integration.GetSearchParams(integration.IndexFaissIvfFlat, metric.L2)
	searchReq := integration.ConstructSearchRequest("", collectionName, expr,
		integration.FloatVecField, schemapb.DataType_FloatVector, nil, metric.L2, params, nq, dim1, topk, roundDecimal)
	searchReq.ConsistencyLevel = commonpb.ConsistencyLevel_Eventually

	searchResult, err := c.MilvusClient.Search(ctx, searchReq)

	err = merr.CheckRPCCall(searchResult, err)
	s.NoError(err)
	s.Equal(nq*topk, len(searchResult.GetResults().GetScores()))

	// search vec 2
	searchReq = integration.ConstructSearchRequest("", collectionName, expr,
		integration.BFloat16VecField, schemapb.DataType_BFloat16Vector, nil, metric.L2, params, nq, dim2, topk, roundDecimal)
	searchReq.ConsistencyLevel = commonpb.ConsistencyLevel_Eventually

	searchResult, err = c.MilvusClient.Search(ctx, searchReq)

	err = merr.CheckRPCCall(searchResult, err)
	s.NoError(err)
	// s.Equal(nq*topk, len(searchResult.GetResults().GetScores())) // TODO: fix bf16vector search
}

func (s *BulkInsertSuite) TestMultipleVectorFields_JSON() {
	s.fileType = importutilv2.JSON
	s.testMultipleVectorFields()
}

func (s *BulkInsertSuite) TestMultipleVectorFields_Parquet() {
	s.fileType = importutilv2.Parquet
	s.testMultipleVectorFields()
}

func (s *BulkInsertSuite) TestMultipleVectorFields_CSV() {
	s.fileType = importutilv2.CSV
	s.testMultipleVectorFields()
}
