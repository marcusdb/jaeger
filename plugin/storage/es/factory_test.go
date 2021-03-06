// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package es

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"

	"github.com/jaegertracing/jaeger/pkg/config"
	"github.com/jaegertracing/jaeger/pkg/es"
	escfg "github.com/jaegertracing/jaeger/pkg/es/config"
	"github.com/jaegertracing/jaeger/pkg/es/mocks"
	"github.com/jaegertracing/jaeger/storage"
)

var _ storage.Factory = new(Factory)

type mockClientBuilder struct {
	escfg.Configuration
	err error
}

func (m *mockClientBuilder) NewClient(logger *zap.Logger, metricsFactory metrics.Factory) (es.Client, error) {
	if m.err == nil {
		return &mocks.Client{}, nil
	}
	return nil, m.err
}

func TestElasticsearchFactory(t *testing.T) {
	f := NewFactory()
	v, command := config.Viperize(f.AddFlags)
	command.ParseFlags([]string{})
	f.InitFromViper(v)

	// after InitFromViper, f.primaryConfig points to a real session builder that will fail in unit tests,
	// so we override it with a mock.
	f.primaryConfig = &mockClientBuilder{err: errors.New("made-up error")}
	assert.EqualError(t, f.Initialize(metrics.NullFactory, zap.NewNop()), "made-up error")

	f.primaryConfig = &mockClientBuilder{}
	f.archiveConfig = &mockClientBuilder{err: errors.New("made-up error2")}
	assert.EqualError(t, f.Initialize(metrics.NullFactory, zap.NewNop()), "made-up error2")

	f.archiveConfig = &mockClientBuilder{}
	assert.NoError(t, f.Initialize(metrics.NullFactory, zap.NewNop()))

	_, err := f.CreateSpanReader()
	assert.NoError(t, err)

	_, err = f.CreateSpanWriter()
	assert.NoError(t, err)

	_, err = f.CreateDependencyReader()
	assert.NoError(t, err)

	_, err = f.CreateArchiveSpanReader()
	assert.NoError(t, err)

	_, err = f.CreateArchiveSpanWriter()
	assert.NoError(t, err)
}

func TestElasticsearchTagsFileDoNotExist(t *testing.T) {
	f := NewFactory()
	mockConf := &mockClientBuilder{}
	mockConf.TagsFilePath = "fixtures/tags_foo.txt"
	f.primaryConfig = mockConf
	f.archiveConfig = mockConf
	assert.NoError(t, f.Initialize(metrics.NullFactory, zap.NewNop()))
	r, err := f.CreateSpanWriter()
	require.Error(t, err)
	assert.Nil(t, r)
}

func TestLoadTagsFromFile(t *testing.T) {
	tests := []struct {
		path  string
		tags  []string
		error bool
	}{
		{
			path:  "fixtures/do_not_exists.txt",
			error: true,
		},
		{
			path: "fixtures/tags_01.txt",
			tags: []string{"foo", "bar", "space"},
		},
		{
			path: "fixtures/tags_02.txt",
			tags: nil,
		},
	}

	for _, test := range tests {
		tags, err := loadTagsFromFile(test.path)
		if test.error {
			require.Error(t, err)
			assert.Nil(t, tags)
		} else {
			assert.Equal(t, test.tags, tags)
		}
	}
}
