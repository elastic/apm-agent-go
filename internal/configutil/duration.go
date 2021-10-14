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

package configutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var durationUnitMap = map[string]time.Duration{
	"us": time.Microsecond,
	"ms": time.Millisecond,
	"s":  time.Second,
	"m":  time.Minute,
}

// DurationOptions can be used to specify the minimum accepted duration unit
// for ParseDurationOptions.
type DurationOptions struct {
	MinimumDurationUnit time.Duration
}

// ParseDuration parses s as a duration, accepting a subset
// of the syntax supported by time.ParseDuration.
//
// Valid time units are "ms", "s", "m".
func ParseDuration(s string) (time.Duration, error) {
	return ParseDurationOptions(s, DurationOptions{
		MinimumDurationUnit: time.Millisecond,
	})
}

// ParseDurationOptions parses s as a duration, accepting a subset of the
// syntax supported by time.ParseDuration. It allows a DurationOptions to
// be passed to specify the minimum time.Duration unit allowed.
//
// Valid time units are "us", "ms", "s", "m".
func ParseDurationOptions(s string, opts DurationOptions) (time.Duration, error) {
	orig := s
	mul := time.Nanosecond
	if strings.HasPrefix(s, "-") {
		mul = -1
		s = s[1:]
	}

	sep := -1
	for i, c := range s {
		if sep == -1 {
			if c < '0' || c > '9' {
				sep = i
				break
			}
		}
	}

	allowedUnitsString := computeAllowedUnitsString(
		opts.MinimumDurationUnit, time.Minute,
	)
	if sep == -1 {
		return 0, fmt.Errorf("missing unit in duration %s (allowed units: %s)",
			orig, allowedUnitsString,
		)
	}

	n, err := strconv.ParseInt(s[:sep], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %s", orig)
	}

	// If it's
	mul, ok := durationUnitMap[s[sep:]]
	if ok {
		if mul < opts.MinimumDurationUnit {
			return 0, fmt.Errorf("invalid unit in duration %s (allowed units: %s)",
				orig, allowedUnitsString,
			)
		}
		return mul * time.Duration(n), nil
	}

	for _, c := range s[sep:] {
		if unicode.IsSpace(c) {
			return 0, fmt.Errorf("invalid character %q in duration %s", c, orig)
		}
	}
	return 0, fmt.Errorf("invalid unit in duration %s (allowed units: %s)",
		orig, allowedUnitsString,
	)
}

// computeAllowedUnitsString returns a string
func computeAllowedUnitsString(minUnit, maxUnit time.Duration) string {
	inverseLookup := make(map[time.Duration]string)
	for k, v := range durationUnitMap {
		inverseLookup[v] = k
	}

	if minUnit < time.Microsecond {
		minUnit = time.Microsecond
	}

	allowedUnits := make([]string, 0, 4)
	nextDuration := time.Duration(1000)
	for i := minUnit; i <= maxUnit; i = i * nextDuration {
		if i >= time.Second {
			nextDuration = 60
		}
		allowedUnits = append(allowedUnits, inverseLookup[i])
	}
	return strings.Join(allowedUnits, ", ")
}
