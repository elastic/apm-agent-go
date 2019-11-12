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

package apmhostutil

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/model"
)

func TestCgroupContainerInfoDocker(t *testing.T) {
	container, kubernetes, err := readCgroupContainerInfo(strings.NewReader(`
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
	assert.Nil(t, kubernetes)
	assert.Equal(t, &model.Container{ID: "051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76"}, container)
}

func TestCgroupContainerInfoECS(t *testing.T) {
	container, kubernetes, err := readCgroupContainerInfo(strings.NewReader(`
3:cpuacct:/ecs/eb9d3d0c-8936-42d7-80d8-f82b2f1a629e/7e9139716d9e5d762d22f9f877b87d1be8b1449ac912c025a984750c5dbff157
`[1:]))

	assert.NoError(t, err)
	assert.Nil(t, kubernetes)
	assert.Equal(t, &model.Container{ID: "7e9139716d9e5d762d22f9f877b87d1be8b1449ac912c025a984750c5dbff157"}, container)
}

func TestCgroupContainerInfoCloudFoundryGarden(t *testing.T) {
	container, kubernetes, err := readCgroupContainerInfo(strings.NewReader(`
1:name=systemd:/system.slice/garden.service/garden/70eb4ce5-a065-4401-6990-88ed
`[1:]))

	assert.NoError(t, err)
	assert.Nil(t, kubernetes)
	assert.Equal(t, &model.Container{ID: "70eb4ce5-a065-4401-6990-88ed"}, container)
}

func TestCgroupContainerInfoNonContainer(t *testing.T) {
	container, _, err := readCgroupContainerInfo(strings.NewReader(`
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
	assert.Nil(t, kubernetes)
	assert.Nil(t, container)
}

func TestCgroupContainerInfoDockerSystemd(t *testing.T) {
	container, kubernetes, err := readCgroupContainerInfo(strings.NewReader(`
1:name=systemd:/system.slice/docker-cde7c2bab394630a42d73dc610b9c57415dced996106665d427f6d0566594411.scope
`[1:]))

	assert.NoError(t, err)
	assert.Nil(t, kubernetes)
	assert.Equal(t, &model.Container{ID: "cde7c2bab394630a42d73dc610b9c57415dced996106665d427f6d0566594411"}, container)
}

func TestCgroupContainerInfoNonHex(t *testing.T) {
	// Imaginary future format. We use the last part of the path,
	// trimming legacy prefix/suffix, and check the expected
	// length and runes used.
	container, kubernetes, err := readCgroupContainerInfo(strings.NewReader(`
1:name=systemd:/docker/051e2ee0bce99116029a13df4a9e943137f19f957f38ac02d6bad96f9b700f76/not_hex
`[1:]))

	assert.NoError(t, err)
	assert.Nil(t, kubernetes)
	assert.Nil(t, container)
}

func TestCgroupContainerInfoKubernetes(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)
	container, kubernetes, err := readCgroupContainerInfo(strings.NewReader(`
1:name=systemd:/kubepods/besteffort/pode9b90526-f47d-11e8-b2a5-080027b9f4fb/15aa6e53-b09a-40c7-8558-c6c31e36c88a`[1:]))

	assert.NoError(t, err)
	assert.Equal(t, &model.Container{ID: "15aa6e53-b09a-40c7-8558-c6c31e36c88a"}, container)
	assert.Equal(t, &model.Kubernetes{
		Pod: &model.KubernetesPod{
			UID:  "e9b90526-f47d-11e8-b2a5-080027b9f4fb",
			Name: hostname,
		},
	}, kubernetes)
}

func TestCgroupContainerInfoKubernetesSystemd(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)
	container, kubernetes, err := readCgroupContainerInfo(strings.NewReader(`
1:name=systemd:/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod90d81341_92de_11e7_8cf2_507b9d4141fa.slice/crio-2227daf62df6694645fee5df53c1f91271546a9560e8600a525690ae252b7f63.scope`[1:]))

	assert.NoError(t, err)
	assert.Equal(t, &model.Container{ID: "2227daf62df6694645fee5df53c1f91271546a9560e8600a525690ae252b7f63"}, container)
	assert.Equal(t, &model.Kubernetes{
		Pod: &model.KubernetesPod{
			UID:  "90d81341-92de-11e7-8cf2-507b9d4141fa",
			Name: hostname,
		},
	}, kubernetes)
}
