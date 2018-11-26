package apmhostutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCgroupDockerContainerID(t *testing.T) {
	id, ok, err := cgroupDockerContainerID(strings.NewReader(`
12:devices:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
11:hugetlb:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
10:memory:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
9:freezer:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
8:perf_event:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
7:blkio:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
6:pids:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
5:rdma:/
4:cpuset:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
3:net_cls,net_prio:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
2:cpu,cpuacct:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
1:name=systemd:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76
0::/system.slice/docker.service`[1:]))

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76", id)
}

func TestCgroupDockerContainerIDNonContainer(t *testing.T) {
	id, ok, err := cgroupDockerContainerID(strings.NewReader(`
12:devices:/user.slice
11:hugetlb:/
10:memory:/user.slice
9:freezer:/
8:perf_event:/
7:blkio:/user.slice
6:pids:/user.slice/user-1000.slice/session-2.scope
5:rdma:/
4:cpuset:/
3:net_cls,net_prio:/
2:cpu,cpuacct:/user.slice
1:name=systemd:/user.slice/user-1000.slice/session-2.scope
0::/user.slice/user-1000.slice/session-2.scope`[1:]))

	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", id)
}

func TestCgroupDockerContainerIDLegacy(t *testing.T) {
	id, ok, err := cgroupDockerContainerID(strings.NewReader(`
1:name=systemd:/system.slice/docker-cde7c2bab394630a42d73dc610b9c57415dced996106665d427f6d0566594411.scope
`[1:]))

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "cde7c2bab394630a42d73dc610b9c57415dced996106665d427f6d0566594411", id)
}

func TestCgroupDockerContainerIDNonHex(t *testing.T) {
	// Imaginary future format. We use the last part of the path,
	// trimming legacy prefix/suffix, and check the expected
	// length and runes used.
	id, ok, err := cgroupDockerContainerID(strings.NewReader(`
1:name=systemd:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76/not_hex
`[1:]))

	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", id)
}
