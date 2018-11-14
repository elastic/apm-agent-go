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

package apmzerolog_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmzerolog"
)

func ExampleMarshalErrorStack() {
	zerolog.ErrorStackMarshaler = apmzerolog.MarshalErrorStack

	logger := zerolog.New(os.Stdout)
	logger.Error().Stack().Err(errors.New("aieee")).Msg("nope nope nope")
}

func TestMarshalErrorStack(t *testing.T) {
	zerolog.ErrorStackMarshaler = apmzerolog.MarshalErrorStack

	// Make an error whose stack does not involve the testing framework.
	errch := make(chan error, 1)
	go func() {
		errch <- funcA()
	}()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	logger.Error().Stack().Err(<-errch).Msg("nope nope nope")

	var record struct {
		Level   string
		Message string
		Error   string
		Stack   []struct {
			Func   string
			Line   string
			Source string
		}
	}
	if err := json.NewDecoder(&buf).Decode(&record); err != nil {
		panic(err)
	}

	assert.Equal(t, record.Level, "error")
	assert.Equal(t, record.Message, "nope nope nope")
	assert.Equal(t, record.Error, "error from funcC")

	assert.Equal(t, "go.elastic.co/apm/module/apmzerolog_test.funcC", record.Stack[0].Func)
	assert.Equal(t, "go.elastic.co/apm/module/apmzerolog_test.funcB", record.Stack[1].Func)
	assert.Equal(t, "go.elastic.co/apm/module/apmzerolog_test.funcA", record.Stack[2].Func)
	for _, frame := range record.Stack {
		assert.NotEmpty(t, frame.Source)
		assert.NotEmpty(t, frame.Line)
	}
}

func funcA() error {
	return funcB()
}

func funcB() error {
	return funcC()
}

func funcC() error {
	return errors.New("error from funcC")
}
