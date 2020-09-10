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

package apmtest_test

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
)

func TestRecordingTracerCloudMetadata(t *testing.T) {
	r := apmtest.NewRecordingTracer()

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	fmt.Fprint(zw, `{"metadata":{"cloud":{"provider":"zeus","region":"troposphere","availability_zone":"torrid","account":{"id":"okayenough"}}}}`)
	assert.NoError(t, zw.Close())

	err := r.SendStream(context.Background(), &buf)
	require.NoError(t, err)
	assert.Equal(t, model.Cloud{
		Provider:         "zeus",
		Region:           "troposphere",
		AvailabilityZone: "torrid",
		Account:          &model.CloudAccount{ID: "okayenough"},
	}, r.CloudMetadata())
}
