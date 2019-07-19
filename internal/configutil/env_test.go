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

package configutil_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/internal/configutil"
	"go.elastic.co/apm/internal/wildcard"
)

func TestParseDurationEnv(t *testing.T) {
	const envKey = "ELASTIC_APM_TEST_DURATION"
	os.Unsetenv(envKey)
	defer os.Unsetenv(envKey)

	d, err := configutil.ParseDurationEnv(envKey, 42*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 42*time.Second, d)

	os.Setenv(envKey, "5s")
	d, err = configutil.ParseDurationEnv(envKey, 42*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, d)

	os.Setenv(envKey, "5ms")
	d, err = configutil.ParseDurationEnv(envKey, 42*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Millisecond, d)

	os.Setenv(envKey, "5m")
	d, err = configutil.ParseDurationEnv(envKey, 42*time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Minute, d)

	os.Setenv(envKey, "5 h")
	_, err = configutil.ParseDurationEnv(envKey, 42*time.Second)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_DURATION: invalid character ' ' in duration 5 h")

	os.Setenv(envKey, "5h")
	_, err = configutil.ParseDurationEnv(envKey, 42*time.Second)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_DURATION: invalid unit in duration 5h (allowed units: ms, s, m)")

	os.Setenv(envKey, "5")
	_, err = configutil.ParseDurationEnv(envKey, 42*time.Second)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_DURATION: missing unit in duration 5 (allowed units: ms, s, m)")

	os.Setenv(envKey, "blah")
	_, err = configutil.ParseDurationEnv(envKey, 42*time.Second)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_DURATION: invalid duration blah")
}

func TestParseSizeEnv(t *testing.T) {
	const envKey = "ELASTIC_APM_TEST_SIZE"
	os.Unsetenv(envKey)
	defer os.Unsetenv(envKey)

	d, err := configutil.ParseSizeEnv(envKey, 42*configutil.KByte)
	assert.NoError(t, err)
	assert.Equal(t, 42*configutil.KByte, d)

	os.Setenv(envKey, "5b")
	d, err = configutil.ParseSizeEnv(envKey, 42*configutil.KByte)
	assert.NoError(t, err)
	assert.Equal(t, 5*configutil.Byte, d)

	os.Setenv(envKey, "5kb")
	d, err = configutil.ParseSizeEnv(envKey, 42*configutil.KByte)
	assert.NoError(t, err)
	assert.Equal(t, 5*configutil.KByte, d)

	os.Setenv(envKey, "5mb")
	d, err = configutil.ParseSizeEnv(envKey, 42*configutil.KByte)
	assert.NoError(t, err)
	assert.Equal(t, 5*configutil.MByte, d)

	os.Setenv(envKey, "5gb")
	d, err = configutil.ParseSizeEnv(envKey, 42*configutil.KByte)
	assert.NoError(t, err)
	assert.Equal(t, 5*configutil.GByte, d)

	os.Setenv(envKey, "5GB")
	d, err = configutil.ParseSizeEnv(envKey, 42*configutil.KByte)
	assert.NoError(t, err)
	assert.Equal(t, 5*configutil.GByte, d)

	os.Setenv(envKey, "5 mb")
	_, err = configutil.ParseSizeEnv(envKey, 0)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_SIZE: invalid character ' ' in size 5 mb")

	os.Setenv(envKey, "5tb")
	_, err = configutil.ParseSizeEnv(envKey, 0)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_SIZE: invalid unit in size 5tb (allowed units: B, KB, MB, GB)")

	os.Setenv(envKey, "5")
	_, err = configutil.ParseSizeEnv(envKey, 0)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_SIZE: missing unit in size 5 (allowed units: B, KB, MB, GB)")

	os.Setenv(envKey, "blah")
	_, err = configutil.ParseSizeEnv(envKey, 0)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_SIZE: invalid size blah")
}

func TestParseBoolEnv(t *testing.T) {
	const envKey = "ELASTIC_APM_TEST_BOOL"
	os.Unsetenv(envKey)
	defer os.Unsetenv(envKey)

	b, err := configutil.ParseBoolEnv(envKey, true)
	assert.NoError(t, err)
	assert.True(t, b)

	os.Setenv(envKey, "true")
	b, err = configutil.ParseBoolEnv(envKey, false)
	assert.NoError(t, err)
	assert.True(t, b)

	os.Setenv(envKey, "false")
	b, err = configutil.ParseBoolEnv(envKey, true)
	assert.NoError(t, err)
	assert.False(t, b)

	os.Setenv(envKey, "falsk")
	_, err = configutil.ParseBoolEnv(envKey, true)
	assert.EqualError(t, err, `failed to parse ELASTIC_APM_TEST_BOOL: strconv.ParseBool: parsing "falsk": invalid syntax`)
}

func TestParseListEnv(t *testing.T) {
	const envKey = "ELASTIC_APM_TEST_LIST"
	os.Unsetenv(envKey)
	defer os.Unsetenv(envKey)

	defaultList := []string{"foo", "bar"}

	list := configutil.ParseListEnv(envKey, ",", defaultList)
	assert.Equal(t, defaultList, list)

	os.Setenv(envKey, "a")
	list = configutil.ParseListEnv(envKey, ",", defaultList)
	assert.Equal(t, []string{"a"}, list)

	os.Setenv(envKey, "a,b")
	list = configutil.ParseListEnv(envKey, ",", defaultList)
	assert.Equal(t, []string{"a", "b"}, list)

	os.Setenv(envKey, ",a , b,")
	list = configutil.ParseListEnv(envKey, ",", defaultList)
	assert.Equal(t, []string{"a", "b"}, list)

	os.Setenv(envKey, ",")
	list = configutil.ParseListEnv(envKey, ",", defaultList)
	assert.Len(t, list, 0)

	os.Setenv(envKey, "a| b")
	list = configutil.ParseListEnv(envKey, "|", defaultList)
	assert.Equal(t, []string{"a", "b"}, list)

	os.Setenv(envKey, "a b")
	list = configutil.ParseListEnv(envKey, ",", defaultList)
	assert.Equal(t, []string{"a b"}, list)
}

func TestParseWildcardPatternsEnv(t *testing.T) {
	const envKey = "ELASTIC_APM_TEST_WILDCARDS"
	os.Unsetenv(envKey)
	defer os.Unsetenv(envKey)

	newMatchers := func(p ...string) wildcard.Matchers {
		matchers := make(wildcard.Matchers, len(p))
		for i, p := range p {
			matchers[i] = wildcard.NewMatcher(p, wildcard.CaseInsensitive)
		}
		return matchers
	}
	defaultMatchers := newMatchers("default")

	matchers := configutil.ParseWildcardPatternsEnv(envKey, defaultMatchers)
	assert.Equal(t, defaultMatchers, matchers)

	os.Setenv(envKey, "foo, bar")
	expected := newMatchers("foo", "bar")
	matchers = configutil.ParseWildcardPatternsEnv(envKey, defaultMatchers)
	assert.Equal(t, expected, matchers)

	os.Setenv(envKey, "foo, (?-i)bar")
	expected[1] = wildcard.NewMatcher("bar", wildcard.CaseSensitive)
	matchers = configutil.ParseWildcardPatternsEnv(envKey, defaultMatchers)
	assert.Equal(t, expected, matchers)

	os.Setenv(envKey, "(?i)foo, (?-i)bar")
	matchers = configutil.ParseWildcardPatternsEnv(envKey, defaultMatchers)
	assert.Equal(t, expected, matchers)
}
