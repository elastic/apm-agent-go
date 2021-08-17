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

//go:build go1.13
// +build go1.13

package apm_test

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/transport/transporttest"
)

func TestErrorCauseUnwrap(t *testing.T) {
	err := fmt.Errorf("%w", errors.New("cause"))

	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.NewError(err).Send()
	tracer.Flush(nil)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	assert.Equal(t, "TestErrorCauseUnwrap", payloads.Errors[0].Culprit)

	require.Len(t, payloads.Errors[0].Exception.Cause, 1)
	assert.Equal(t, "cause", payloads.Errors[0].Exception.Cause[0].Message)
}
