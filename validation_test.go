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

package apm_test

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/internal/apmschema"
)

func TestValidateServiceName(t *testing.T) {
	validatePayloadMetadata(t, func(tracer *apm.Tracer) {
		tracer.Service.Name = strings.Repeat("x", 1025)
	})
}

func TestValidateServiceVersion(t *testing.T) {
	validatePayloadMetadata(t, func(tracer *apm.Tracer) {
		tracer.Service.Version = strings.Repeat("x", 1025)
	})
}

func TestValidateServiceEnvironment(t *testing.T) {
	validatePayloadMetadata(t, func(tracer *apm.Tracer) {
		tracer.Service.Environment = strings.Repeat("x", 1025)
	})
}

func TestValidateTransactionName(t *testing.T) {
	validatePayloads(t, func(tracer *apm.Tracer) {
		tracer.StartTransaction(strings.Repeat("x", 1025), "type").End()
	})
}

func TestValidateTransactionType(t *testing.T) {
	validatePayloads(t, func(tracer *apm.Tracer) {
		tracer.StartTransaction("name", strings.Repeat("x", 1025)).End()
	})
}

func TestValidateTransactionResult(t *testing.T) {
	validatePayloads(t, func(tracer *apm.Tracer) {
		tx := tracer.StartTransaction("name", "type")
		tx.Result = strings.Repeat("x", 1025)
		tx.End()
	})
}

func TestValidateSpanName(t *testing.T) {
	validateTransaction(t, func(tx *apm.Transaction) {
		tx.StartSpan(strings.Repeat("x", 1025), "type", nil).End()
	})
}

func TestValidateSpanType(t *testing.T) {
	validateTransaction(t, func(tx *apm.Transaction) {
		tx.StartSpan("name", strings.Repeat("x", 1025), nil).End()
	})
}

func TestValidateDatabaseSpanContext(t *testing.T) {
	validateSpan(t, func(s *apm.Span) {
		s.Context.SetDatabase(apm.DatabaseSpanContext{
			Instance:  strings.Repeat("x", 1025),
			Statement: strings.Repeat("x", 1025),
			Type:      strings.Repeat("x", 1025),
			User:      strings.Repeat("x", 1025),
		})
	})
}

func TestValidateDestinationSpanContext(t *testing.T) {
	validateSpan(t, func(s *apm.Span) {
		s.Context.SetDestinationAddress(strings.Repeat("x", 1025), 0)
		s.Context.SetDestinationService(apm.DestinationServiceSpanContext{
			Name:     strings.Repeat("x", 1025),
			Resource: strings.Repeat("x", 1025),
		})
	})
}

func TestValidateContextUser(t *testing.T) {
	validateTransaction(t, func(tx *apm.Transaction) {
		tx.Context.SetUsername(strings.Repeat("x", 1025))
		tx.Context.SetUserEmail(strings.Repeat("x", 1025))
		tx.Context.SetUserID(strings.Repeat("x", 1025))
	})
}

func TestValidateContextUserBasicAuth(t *testing.T) {
	validateTransaction(t, func(tx *apm.Transaction) {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		req.SetBasicAuth(strings.Repeat("x", 1025), "")
		tx.Context.SetHTTPRequest(req)
	})
}

func TestValidateContextLabels(t *testing.T) {
	t.Run("long_key", func(t *testing.T) {
		// NOTE(axw) this should probably fail, but does not. See:
		// https://github.com/elastic/apm-server/issues/910
		validateTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetLabel(strings.Repeat("x", 1025), "x")
		})
	})
	t.Run("long_value", func(t *testing.T) {
		validateTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetLabel("x", strings.Repeat("x", 1025))
		})
	})
	t.Run("reserved_key_chars", func(t *testing.T) {
		validateTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetLabel("x.y", "z")
		})
	})
	t.Run("null_value", func(t *testing.T) {
		validateTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetLabel("null", nil)
		})
	})
}

func TestValidateContextCustom(t *testing.T) {
	t.Run("long_value", func(t *testing.T) {
		validateTransaction(t, func(tx *apm.Transaction) {
			// Context values are not indexed, so they're not truncated.
			tx.Context.SetCustom("x", strings.Repeat("x", 1025))
		})
	})
	t.Run("reserved_key_chars", func(t *testing.T) {
		validateTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetCustom("x.y", "z")
		})
	})
}

func TestValidateRequestMethod(t *testing.T) {
	validateTransaction(t, func(tx *apm.Transaction) {
		req, _ := http.NewRequest(strings.Repeat("x", 1025), "/", nil)
		tx.Context.SetHTTPRequest(req)
	})
}

func TestValidateRequestBody(t *testing.T) {
	t.Run("raw", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			tracer.SetCaptureBody(apm.CaptureBodyAll)
			tx := tracer.StartTransaction("name", "type")
			defer tx.End()

			body := strings.NewReader(strings.Repeat("x", 1025))
			req, _ := http.NewRequest("GET", "/", body)
			captureBody := tracer.CaptureHTTPRequestBody(req)
			tx.Context.SetHTTPRequest(req)
			tx.Context.SetHTTPRequestBody(captureBody)
		})
	})
	t.Run("form", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			tracer.SetCaptureBody(apm.CaptureBodyAll)
			tx := tracer.StartTransaction("name", "type")
			defer tx.End()

			req, _ := http.NewRequest("GET", "/", strings.NewReader("x"))
			req.PostForm = url.Values{
				"unsanitized_field": []string{strings.Repeat("x", 1025)},
			}
			captureBody := tracer.CaptureHTTPRequestBody(req)
			tx.Context.SetHTTPRequest(req)
			tx.Context.SetHTTPRequestBody(captureBody)
		})
	})
}

func TestValidateRequestURL(t *testing.T) {
	type test struct {
		name string
		url  string
	}
	long := strings.Repeat("x", 1025)
	longNumber := strings.Repeat("8", 1025)
	tests := []test{
		{name: "scheme", url: fmt.Sprintf("%s://testing.invalid", long)},
		{name: "hostname", url: fmt.Sprintf("http://%s/", long)},
		{name: "port", url: fmt.Sprintf("http://testing.invalid:%s/", longNumber)},
		{name: "path", url: fmt.Sprintf("http://testing.invalid/%s", long)},
		{name: "query", url: fmt.Sprintf("http://testing.invalid/?%s", long)},
		{name: "fragment", url: fmt.Sprintf("http://testing.invalid/#%s", long)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validateTransaction(t, func(tx *apm.Transaction) {
				req, _ := http.NewRequest("GET", test.url, nil)
				tx.Context.SetHTTPRequest(req)
			})
		})
	}
}

func TestValidateErrorException(t *testing.T) {
	t.Run("empty_message", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			tracer.NewError(&testError{
				message: "",
			}).Send()
		})
	})
	t.Run("long_message", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			tracer.NewError(&testError{
				message: strings.Repeat("x", 1025),
			}).Send()
		})
	})
	t.Run("culprit", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			e := tracer.NewError(&testError{message: "foo"})
			e.Culprit = strings.Repeat("x", 1025)
			e.Send()
		})
	})
	t.Run("code", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			tracer.NewError(&testError{
				message: "xyz",
				code:    strings.Repeat("x", 1025),
			}).Send()
		})
	})
	t.Run("type", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			tracer.NewError(&testError{
				message: "xyz",
				type_:   strings.Repeat("x", 1025),
			}).Send()
		})
	})
	t.Run("chained", func(t *testing.T) {
		validatePayloads(t, func(tracer *apm.Tracer) {
			tracer.NewError(&testError{
				message: "e1",
				cause: &testError{
					message: "e2",
					cause:   errors.New("hullo"),
				},
			}).Send()
		})
	})
}

func TestValidateErrorLog(t *testing.T) {
	tests := map[string]apm.ErrorLogRecord{
		"empty_message": {
			Message: "",
		},
		"long_message": {
			Message: strings.Repeat("x", 1025),
		},
		"level": {
			Message: "x",
			Level:   strings.Repeat("x", 1025),
		},
		"logger_name": {
			Message:    "x",
			LoggerName: strings.Repeat("x", 1025),
		},
		"message_format": {
			Message:       "x",
			MessageFormat: strings.Repeat("x", 1025),
		},
	}
	for name, record := range tests {
		t.Run(name, func(t *testing.T) {
			validatePayloads(t, func(tracer *apm.Tracer) {
				tracer.NewErrorLog(record).Send()
			})
		})
	}
}

func TestValidateMetrics(t *testing.T) {
	gather := func(ctx context.Context, m *apm.Metrics) error {
		m.Add("without_labels", nil, -66)
		m.Add("with_labels", []apm.MetricLabel{
			{Name: "name", Value: "value"},
		}, -66)
		return nil
	}

	validatePayloads(t, func(tracer *apm.Tracer) {
		unregister := tracer.RegisterMetricsGatherer(apm.GatherMetricsFunc(gather))
		defer unregister()
		tracer.SendMetrics(nil)
	})
}

func validateSpan(t *testing.T, f func(s *apm.Span)) {
	validateTransaction(t, func(tx *apm.Transaction) {
		s := tx.StartSpan("name", "type", nil)
		f(s)
		s.End()
	})
}

func validateTransaction(t *testing.T, f func(tx *apm.Transaction)) {
	validatePayloads(t, func(tracer *apm.Tracer) {
		tx := tracer.StartTransaction("name", "type")
		f(tx)
		tx.End()
	})
}

func validatePayloadMetadata(t *testing.T, f func(tracer *apm.Tracer)) {
	validatePayloads(t, func(tracer *apm.Tracer) {
		f(tracer)
		tracer.StartTransaction("name", "type").End()
	})
}

func validatePayloads(t *testing.T, f func(tracer *apm.Tracer)) {
	tracer, _ := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:        "x",
		ServiceVersion:     "y",
		ServiceEnvironment: "z",
		Transport:          &validatingTransport{t: t},
	})
	defer tracer.Close()

	f(tracer)
	tracer.Flush(nil)
}

type validatingTransport struct {
	t *testing.T
}

func (t *validatingTransport) SendStream(ctx context.Context, r io.Reader) error {
	zr, err := zlib.NewReader(r)
	require.NoError(t.t, err)

	first := true
	s := bufio.NewScanner(zr)
	lineno := 0
	for s.Scan() {
		lineno++
		m := make(map[string]json.RawMessage)
		err := json.Unmarshal(s.Bytes(), &m)
		require.NoError(t.t, err)
		require.Len(t.t, m, 1) // 1 object per line

		var schema *jsonschema.Schema
		for k, v := range m {
			if first {
				require.Equal(t.t, "metadata", k)
				first = false
				schema = apmschema.Metadata
			} else {
				switch k {
				case "error":
					schema = apmschema.Error
				case "metricset":
					schema = apmschema.MetricSet
				case "span":
					schema = apmschema.Span
				case "transaction":
					schema = apmschema.Transaction
				default:
					t.t.Errorf("invalid object %q on line %d", k, lineno)
					continue
				}
			}
			err := schema.Validate(bytes.NewReader([]byte(v)))
			if assert.NoError(t.t, err) {
				// Perform additional validation.
				t.checkStringLengths(k, v)
			}
		}
	}
	assert.NoError(t.t, s.Err())
	if first {
		t.t.Errorf("metadata missing from stream")
	}
	return nil
}

func (t *validatingTransport) checkStringLengths(path string, raw json.RawMessage) {
	// No string should exceed 10000 runes.
	const limit = 10000
	var checkRecursive func(path string, v interface{}) bool
	checkRecursive = func(path string, v interface{}) bool {
		switch v := v.(type) {
		case map[string]interface{}:
			for k, v := range v {
				path := path + "." + k
				if !checkRecursive(path, v) {
					return false
				}
			}
		case string:
			n := utf8.RuneCountInString(v)
			if !assert.Condition(t.t, func() bool {
				return n <= limit
			}, "len(%s) > %d (==%d)", path, limit, n) {
				return false
			}
		case []interface{}:
			for i, v := range v {
				path := fmt.Sprintf("%s[%d]", path, i)
				if !checkRecursive(path, v) {
					return false
				}
			}
		}
		return true
	}
	m := make(map[string]interface{})
	if err := json.Unmarshal(raw, &m); !assert.NoError(t.t, err) {
		return
	}
	checkRecursive(path, m)
}

type testError struct {
	message string
	code    string
	type_   string
	cause   error
}

func (e *testError) Error() string {
	return e.message
}

func (e *testError) Code() string {
	return e.code
}

func (e *testError) Type() string {
	return e.type_
}

func (e *testError) Cause() error {
	return e.cause
}
