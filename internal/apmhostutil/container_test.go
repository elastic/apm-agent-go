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
		// Currently we only support ContainerID in Linux containers.
		_, err := apmhostutil.ContainerID()
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

	id, err := apmhostutil.ContainerID()
	require.NoError(t, err)
	assert.Len(t, id, 64)

	// Docker sets the container hostname to a prefix
	// of the full container ID.
	hostname, err := os.Hostname()
	require.NoError(t, err)
	assert.Equal(t, hostname, id[:len(hostname)])
}
