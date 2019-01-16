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

package apmlog

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Unsetenv("ELASTIC_APM_LOG_FILE")
	os.Unsetenv("ELASTIC_APM_LOG_LEVEL")
	DefaultLogger = nil
}

func TestInitDefaultLoggerNoEnv(t *testing.T) {
	DefaultLogger = nil
	initDefaultLogger()
	assert.Nil(t, DefaultLogger)
}

func TestInitDefaultLoggerInvalidFile(t *testing.T) {
	var logbuf bytes.Buffer
	log.SetOutput(&logbuf)

	DefaultLogger = nil
	os.Setenv("ELASTIC_APM_LOG_FILE", ".")
	defer os.Unsetenv("ELASTIC_APM_LOG_FILE")
	initDefaultLogger()

	assert.Nil(t, DefaultLogger)
	assert.Regexp(t, `failed to create "\.": .* \(disabling logging\)`, logbuf.String())
}

func TestInitDefaultLoggerFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	DefaultLogger = nil
	os.Setenv("ELASTIC_APM_LOG_FILE", filepath.Join(dir, "log.json"))
	defer os.Unsetenv("ELASTIC_APM_LOG_FILE")
	initDefaultLogger()

	require.NotNil(t, DefaultLogger)
	DefaultLogger.Debugf("debug message")
	DefaultLogger.Errorf("error message")

	data, err := ioutil.ReadFile(filepath.Join(dir, "log.json"))
	require.NoError(t, err)
	assert.Regexp(t, `{"level":"error","time":".*","message":"error message"}`, string(data))
}

func TestInitDefaultLoggerStdio(t *testing.T) {
	origStdout, origStderr := os.Stdout, os.Stderr
	defer func() {
		os.Stdout, os.Stderr = origStdout, origStderr
	}()

	tempStdout, err := ioutil.TempFile("", "stdout-")
	require.NoError(t, err)
	defer os.Remove(tempStdout.Name())
	defer tempStdout.Close()
	os.Stdout = tempStdout

	tempStderr, err := ioutil.TempFile("", "stderr-")
	require.NoError(t, err)
	defer os.Remove(tempStdout.Name())
	defer tempStderr.Close()
	os.Stderr = tempStderr

	defer os.Unsetenv("ELASTIC_APM_LOG_FILE")
	for _, filename := range []string{"stdout", "stderr"} {
		DefaultLogger = nil
		os.Setenv("ELASTIC_APM_LOG_FILE", filename)
		initDefaultLogger()
		require.NotNil(t, DefaultLogger)
		DefaultLogger.Errorf("%s", filename)
	}

	stdoutContents, err := ioutil.ReadFile(tempStdout.Name())
	require.NoError(t, err)
	assert.Regexp(t, `{"level":"error","time":".*","message":"stdout"}`, string(stdoutContents))

	stderrContents, err := ioutil.ReadFile(tempStderr.Name())
	require.NoError(t, err)
	assert.Regexp(t, `{"level":"error","time":".*","message":"stderr"}`, string(stderrContents))
}

func TestInitDefaultLoggerInvalidLevel(t *testing.T) {
	var logbuf bytes.Buffer
	log.SetOutput(&logbuf)

	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	DefaultLogger = nil
	os.Setenv("ELASTIC_APM_LOG_FILE", filepath.Join(dir, "log.json"))
	os.Setenv("ELASTIC_APM_LOG_LEVEL", "panic")
	defer os.Unsetenv("ELASTIC_APM_LOG_FILE")
	defer os.Unsetenv("ELASTIC_APM_LOG_LEVEL")
	initDefaultLogger()

	require.NotNil(t, DefaultLogger)
	DefaultLogger.Debugf("debug message")
	DefaultLogger.Errorf("error message")

	data, err := ioutil.ReadFile(filepath.Join(dir, "log.json"))
	require.NoError(t, err)
	assert.Regexp(t, `{"level":"error","time":".*","message":"error message"}`, string(data))
	assert.Regexp(t, `invalid ELASTIC_APM_LOG_LEVEL "panic", falling back to "error"`, logbuf.String())
}

func TestInitDefaultLoggerLevel(t *testing.T) {
	var logbuf bytes.Buffer
	log.SetOutput(&logbuf)

	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	DefaultLogger = nil
	os.Setenv("ELASTIC_APM_LOG_FILE", filepath.Join(dir, "log.json"))
	os.Setenv("ELASTIC_APM_LOG_LEVEL", "debug")
	defer os.Unsetenv("ELASTIC_APM_LOG_FILE")
	defer os.Unsetenv("ELASTIC_APM_LOG_LEVEL")
	initDefaultLogger()

	require.NotNil(t, DefaultLogger)
	DefaultLogger.Debugf("debug message")
	DefaultLogger.Errorf("error message")

	data, err := ioutil.ReadFile(filepath.Join(dir, "log.json"))
	require.NoError(t, err)
	assert.Regexp(t, `
{"level":"debug","time":".*","message":"debug message"}
{"level":"error","time":".*","message":"error message"}`[1:],
		string(data))
}

func BenchmarkDefaultLogger(b *testing.B) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(b, err)
	defer os.RemoveAll(dir)

	DefaultLogger = nil
	os.Setenv("ELASTIC_APM_LOG_FILE", filepath.Join(dir, "log.json"))
	defer os.Unsetenv("ELASTIC_APM_LOG_FILE")
	initDefaultLogger()
	require.NotNil(b, DefaultLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DefaultLogger.Errorf("debug message")
	}
}
