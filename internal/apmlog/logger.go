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

package apmlog

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

var (
	// DefaultLogger is the default Logger to use, if ELASTIC_APM_LOG_* are specified.
	DefaultLogger Logger
)

func init() {
	fileStr := strings.TrimSpace(os.Getenv("ELASTIC_APM_LOG_FILE"))
	if fileStr == "" {
		return
	}

	var logWriter io.Writer
	switch strings.ToLower(fileStr) {
	case "stdout":
		logWriter = os.Stdout
	case "stderr":
		logWriter = os.Stderr
	default:
		f, err := os.Create(fileStr)
		if err != nil {
			log.Printf("failed to create %q: %s (disabling logging)", fileStr, err)
			return
		}
		logWriter = &syncFile{File: f}
	}

	const defaultLevel = zerolog.ErrorLevel
	logger := zerolog.New(logWriter)
	if levelStr := strings.TrimSpace(os.Getenv("ELASTIC_APM_LOG_LEVEL")); levelStr != "" {
		level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
		if err != nil {
			log.Printf("invalid ELASTIC_APM_LOG_LEVEL %q, falling back to %q", levelStr, defaultLevel)
			level = defaultLevel
		}
		logger = logger.Level(level)
	} else {
		logger = logger.Level(defaultLevel)
	}
	DefaultLogger = zerologLogger{l: logger}
}

// Logger provides methods for logging.
type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type zerologLogger struct {
	l zerolog.Logger
}

// Debugf logs a message with log.Printf, with a DEBUG prefix.
func (l zerologLogger) Debugf(format string, args ...interface{}) {
	l.l.Debug().Timestamp().Msgf(format, args...)
}

// Errorf logs a message with log.Printf, with an ERROR prefix.
func (l zerologLogger) Errorf(format string, args ...interface{}) {
	l.l.Error().Timestamp().Msgf(format, args...)
}

type syncFile struct {
	mu sync.Mutex
	*os.File
}

// Write calls f.File.Write with f.mu held, to protect  multiple Tracers
// in the same process from one another.
func (f *syncFile) Write(data []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.File.Write(data)
}
