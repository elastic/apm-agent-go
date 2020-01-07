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

package apm_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/internal/apmgodog"
	"go.elastic.co/apm/model"
	"go.elastic.co/fastjson"
)

var (
	flagDumpMetadata = flag.Bool("dump-metadata", false, "Dump metadata and exit without running any tests")
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	flag.Parse()
	if *flagDumpMetadata {
		dumpMetadata()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestFeatures(t *testing.T) {
	apmgodog.Run(t, []string{"."})
}

func getSubprocessMetadata(t *testing.T, env ...string) (*model.System, *model.Process, *model.Service, model.StringMap) {
	cmd := exec.Command(os.Args[0], "-dump-metadata")
	cmd.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if !assert.NoError(t, cmd.Run()) {
		t.FailNow()
	}

	var system model.System
	var process model.Process
	var service model.Service
	var labels model.StringMap

	output := stdout.String()
	d := json.NewDecoder(&stdout)
	if !assert.NoError(t, d.Decode(&system)) {
		t.Logf("output: %q", output)
		t.FailNow()
	}
	require.NoError(t, d.Decode(&process))
	require.NoError(t, d.Decode(&service))
	require.NoError(t, d.Decode(&labels))
	return &system, &process, &service, labels
}

func dumpMetadata() {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)
	system, process, service, labels := tracer.Metadata()

	var w fastjson.Writer
	for _, m := range []fastjson.Marshaler{&system, &process, &service, labels} {
		if err := m.MarshalFastJSON(&w); err != nil {
			panic(err)
		}
	}
	os.Stdout.Write(w.Bytes())
}
