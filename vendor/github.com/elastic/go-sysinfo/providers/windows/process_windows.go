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

package windows

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/elastic/go-windows"

	"github.com/elastic/go-sysinfo/types"
)

var (
	selfPID = os.Getpid()
)

func (s windowsSystem) Processes() ([]types.Process, error) {
	return nil, types.ErrNotImplemented
}

func (s windowsSystem) Process(pid int) (types.Process, error) {
	if pid == selfPID {
		return s.Self()
	}
	// TODO implement support for enumerating processes.
	return nil, types.ErrNotImplemented
}

func (s windowsSystem) Self() (types.Process, error) {
	return newProcess(selfPID)
}

type process struct {
	pid  int
	info types.ProcessInfo
}

func newProcess(pid int) (*process, error) {
	p := &process{pid: pid}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *process) init() error {
	if p.pid != selfPID {
		// TODO implement support for other processes.
		return types.ErrNotImplemented
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	handle, err := p.open()
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)

	var creationTime, exitTime, kernelTime, userTime syscall.Filetime
	if err := syscall.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime); err != nil {
		return err
	}

	p.info = types.ProcessInfo{
		Name:      filepath.Base(exe),
		PID:       selfPID,
		PPID:      os.Getppid(),
		CWD:       cwd,
		Exe:       exe,
		Args:      os.Args[:],
		StartTime: time.Unix(0, creationTime.Nanoseconds()),
	}
	return nil
}

func (p *process) open() (syscall.Handle, error) {
	if p.pid != selfPID {
		// TODO implement support for other processes.
		return 0, types.ErrNotImplemented
	}
	return syscall.GetCurrentProcess()
}

func (p *process) Info() (types.ProcessInfo, error) {
	return p.info, nil
}

func (p *process) Memory() (types.MemoryInfo, error) {
	handle, err := p.open()
	if err != nil {
		return types.MemoryInfo{}, err
	}
	defer syscall.CloseHandle(handle)

	counters, err := windows.GetProcessMemoryInfo(handle)
	if err != nil {
		return types.MemoryInfo{}, err
	}

	return types.MemoryInfo{
		Resident: uint64(counters.WorkingSetSize),
		Virtual:  uint64(counters.PrivateUsage),
	}, nil
}

func (p *process) CPUTime() (types.CPUTimes, error) {
	handle, err := p.open()
	if err != nil {
		return types.CPUTimes{}, err
	}
	defer syscall.CloseHandle(handle)

	var creationTime, exitTime, kernelTime, userTime syscall.Filetime
	if err := syscall.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime); err != nil {
		return types.CPUTimes{}, err
	}

	return types.CPUTimes{
		User:   windows.FiletimeToDuration(&userTime),
		System: windows.FiletimeToDuration(&kernelTime),
	}, nil
}
