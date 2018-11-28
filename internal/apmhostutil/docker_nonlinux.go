// +build !linux

package apmhostutil

import (
	"runtime"

	"github.com/pkg/errors"
)

func dockerContainerID() (string, error) {
	return "", errors.Errorf("docker container ID computation not implemented for %s", runtime.GOOS)
}
