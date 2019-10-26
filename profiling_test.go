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
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/apmtest"
)

func TestTracerCPUProfiling(t *testing.T) {
	os.Setenv("ELASTIC_APM_CPU_PROFILE_INTERVAL", "100ms")
	os.Setenv("ELASTIC_APM_CPU_PROFILE_DURATION", "1s")
	defer os.Unsetenv("ELASTIC_APM_CPU_PROFILE_INTERVAL")
	defer os.Unsetenv("ELASTIC_APM_CPU_PROFILE_DURATION")

	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	timeout := time.After(10 * time.Second)
	var profiles [][]byte
	for len(profiles) == 0 {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for profile")
		default: // busy loop so we get some CPU samples
		}
		profiles = tracer.Payloads().Profiles
	}

	info := parseProfile(profiles[0])
	assert.EqualValues(t, []string{"samples/count", "cpu/nanoseconds"}, info.sampleTypes)
}

func TestTracerHeapProfiling(t *testing.T) {
	os.Setenv("ELASTIC_APM_HEAP_PROFILE_INTERVAL", "100ms")
	defer os.Unsetenv("ELASTIC_APM_HEAP_PROFILE_INTERVAL")

	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	timeout := time.After(10 * time.Second)
	var profiles [][]byte

	tick := time.Tick(50 * time.Millisecond)
	for len(profiles) == 0 {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for profile")
		case <-tick:
		}
		profiles = tracer.Payloads().Profiles
	}

	info := parseProfile(profiles[0])
	assert.EqualValues(t, []string{
		"alloc_objects/count", "alloc_space/bytes",
		"inuse_objects/count", "inuse_space/bytes",
	}, info.sampleTypes)
}

// parseProfile parses the profile data using "go tool pprof".
//
// We could use github.com/google/pprof, but prefer not to add
// a dependency for users just to run these unit test. More
// thorough integration testing should be performed elsewhere.
func parseProfile(data []byte) profileInfo {
	f, err := ioutil.TempFile("", "apm_profiletest")
	if err != nil {
		panic(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		panic(err)
	}

	cmd := exec.Command("go", "tool", "pprof", "-raw", f.Name())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var info profileInfo
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if scanner.Text() == "Samples:" && scanner.Scan() {
			info.sampleTypes = strings.Fields(scanner.Text())
			return info
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	panic("failed to locate sample types")
}

type profileInfo struct {
	sampleTypes []string
}
