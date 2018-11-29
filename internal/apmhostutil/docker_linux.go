// +build linux

package apmhostutil

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func dockerContainerID() (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer f.Close()

	id, ok, err := cgroupDockerContainerID(f)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("could not determine Docker container ID")
	}
	return id, nil
}

func cgroupDockerContainerID(r io.Reader) (string, bool, error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		fields := strings.SplitN(s.Text(), ":", 3)
		if len(fields) != 3 {
			continue
		}
		cgroupPath := fields[2]
		if !strings.Contains(cgroupPath, "docker") {
			continue
		}

		// Legacy: /system.slice/docker-<CID>.scope
		// Current: /docker/<CID>
		id := filepath.Base(cgroupPath)
		id = strings.TrimPrefix(id, "docker-")
		id = strings.TrimSuffix(id, ".scope")

		// Sanity check the ID.
		if len(id) != 64 {
			return "", false, nil
		}
		for _, r := range id {
			if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
				return "", false, nil
			}
		}
		return id, true, nil
	}
	if err := s.Err(); err != nil {
		return "", false, err
	}
	return "", false, nil
}
