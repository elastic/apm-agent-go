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

//go:build go1.20
// +build go1.20

package apm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/apmtest"
)

func TestErrorCauseUnwrapJoined(t *testing.T) {
	err := errors.Join(errors.New("cause1"), errors.New("cause2"))
	_, _, errors := apmtest.WithTransaction(func(ctx context.Context) {
		apm.CaptureError(ctx, err).Send()
	})
	require.Len(t, errors, 1)
	require.Len(t, errors[0].Exception.Cause, 2)
	assert.Equal(t, "cause1", errors[0].Exception.Cause[0].Message)
	assert.Equal(t, "cause2", errors[0].Exception.Cause[1].Message)
}
