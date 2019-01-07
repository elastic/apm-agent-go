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

package apmhostutil_test

import (
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/internal/apmhostutil"
)

func TestContainerID(t *testing.T) {
	if runtime.GOOS != "linux" {
		// Currently we only support Container in Linux containers.
		_, err := apmhostutil.Container()
		assert.Error(t, err)
		return
	}

	// Ideally we would have the CI script pass the
	// docker container ID into the test process,
	// but this would make things convoluted. Instead,
	// just do a basic check of cgroup to see if it's
	// Docker-enabled, and then compare the ID to
	// the container hostname.
	data, err := ioutil.ReadFile("/proc/self/cgroup")
	if err != nil {
		t.Skipf("failed to read cgroup (%s)", err)
	}
	if !strings.Contains(string(data), "docker") {
		t.Skipf("not running inside docker")
	}

	container, err := apmhostutil.Container()
	require.NoError(t, err)
	require.NotNil(t, container)
	assert.Len(t, container.ID, 64)

	// Docker sets the container hostname to a prefix
	// of the full container ID.
	hostname, err := os.Hostname()
	require.NoError(t, err)
	assert.Equal(t, hostname, container.ID[:len(hostname)])
}
