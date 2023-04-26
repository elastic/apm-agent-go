// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build go1.18
// +build go1.18

package apmotel

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/transport/transporttest"
)

func TestNewTracerProviderConfig(t *testing.T) {
	apmTracer, _ := transporttest.NewRecorderTracer()

	testCases := []struct {
		name       string
		options    []TracerProviderOption
		wantConfig tracerProviderConfig
	}{
		{
			name:    "Default",
			options: nil,
			wantConfig: tracerProviderConfig{
				apmTracer: apm.DefaultTracer(),
			},
		},
		{
			name: "WithAPMTracer",
			options: []TracerProviderOption{
				WithAPMTracer(apmTracer),
			},
			wantConfig: tracerProviderConfig{
				apmTracer: apmTracer,
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTracerProviderConfig(tt.options...)
			assert.Equal(t, tt.wantConfig, cfg)
		})
	}
}
