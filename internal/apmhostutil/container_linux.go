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

//go:build linux
// +build linux

package apmhostutil

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"go.elastic.co/apm/v2/model"
)

var (
	cgroupContainerInfoOnce  sync.Once
	cgroupContainerInfoError error
	kubernetes               *model.Kubernetes
	container                *model.Container

	kubepodsRegexp = regexp.MustCompile(
		"" +
			`(?:^/kubepods[\S]*/pod([^/]+)$)|` +
			`(?:kubepods[^/]*-pod([^/]+)\.slice)`,
	)

	containerIDRegexp = regexp.MustCompile(
		"" +
			"^[[:xdigit:]]{64}$|" +
			"^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4,}$|" +
			"^[[:xdigit:]]{32}-[[:digit:]]{10}$",
	)
)

func containerInfo() (*model.Container, error) {
	container, _, err := cgroupContainerInfo()
	return container, err
}

func kubernetesInfo() (*model.Kubernetes, error) {
	_, kubernetes, err := cgroupContainerInfo()
	if err == nil && kubernetes == nil {
		return nil, errors.New("could not determine kubernetes info")
	}
	return kubernetes, err
}

func cgroupContainerInfo() (*model.Container, *model.Kubernetes, error) {
	cgroupContainerInfoOnce.Do(func() {
		cgroupContainerInfoError = func() error {
			f, err := os.Open("/proc/self/cgroup")
			if err != nil {
				return err
			}
			defer f.Close()

			c, k, err := readCgroupContainerInfo(f)
			if err != nil {
				return err
			}
			if c == nil {
				return errors.New("could not determine container info")
			}
			container = c
			kubernetes = k
			return nil
		}()
	})
	return container, kubernetes, cgroupContainerInfoError
}

func readCgroupContainerInfo(r io.Reader) (*model.Container, *model.Kubernetes, error) {
	var container *model.Container
	var kubernetes *model.Kubernetes
	s := bufio.NewScanner(r)
	for s.Scan() {
		// split the line according to the format "hierarchy-ID:controller-list:cgroup-path"
		fields := strings.SplitN(s.Text(), ":", 3)
		if len(fields) != 3 {
			continue
		}

		// extract cgroup-path
		cgroupPath := fields[2]

		// split based on the last occurrence of the colon character, if such exists, in order
		// to support paths of containers created by containerd-cri, where the path part takes
		// the form: <dirname>:cri-containerd:<container-ID>
		idx := strings.LastIndex(cgroupPath, ":")
		if idx == -1 {
			// if colon char is not found within the path, the split is done based on the
			// last occurrence of the slash character
			if idx = strings.LastIndex(cgroupPath, "/"); idx == -1 {
				continue
			}
		}

		dirname, basename := cgroupPath[:idx], cgroupPath[idx+1:]

		// If the basename ends with ".scope", check for a hyphen and remove everything up to
		// and including that. This allows us to match .../docker-<container-id>.scope as well
		// as .../<container-id>.
		if strings.HasSuffix(basename, ".scope") {
			basename = strings.TrimSuffix(basename, ".scope")

			if hyphen := strings.Index(basename, "-"); hyphen != -1 {
				basename = basename[hyphen+1:]
			}
		}
		if match := kubepodsRegexp.FindStringSubmatch(dirname); match != nil {
			// By default, Kubernetes will set the hostname of
			// the pod containers to the pod name. Users that
			// override the name should use the Downard API to
			// override the pod name.
			hostname, _ := os.Hostname()
			uid := match[1]
			if uid == "" {
				// Systemd cgroup driver is being used,
				// so we need to unescape '_' back to '-'.
				uid = strings.Replace(match[2], "_", "-", -1)
			}
			kubernetes = &model.Kubernetes{
				Pod: &model.KubernetesPod{
					Name: hostname,
					UID:  uid,
				},
			}
			// We don't check the contents of the last path segment
			// when we've matched "^/kubepods"; we assume that it is
			// a valid container ID.
			container = &model.Container{ID: basename}
		} else if containerIDRegexp.MatchString(basename) {
			container = &model.Container{ID: basename}
		}
	}
	if err := s.Err(); err != nil {
		return nil, nil, err
	}
	return container, kubernetes, nil
}
