package elasticapm_test

import (
	"bytes"
	"context"
	"fmt"
	"go/build"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/internal/fastjson"
	"github.com/elastic/apm-agent-go/model"
)

func TestValidateServiceName(t *testing.T) {
	validatePayloadMetadata(t, func(tracer *elasticapm.Tracer) {
		tracer.Service.Name = strings.Repeat("x", 1025)
	})
}

func TestValidateServiceVersion(t *testing.T) {
	validatePayloadMetadata(t, func(tracer *elasticapm.Tracer) {
		tracer.Service.Version = strings.Repeat("x", 1025)
	})
}

func TestValidateServiceEnvironment(t *testing.T) {
	validatePayloadMetadata(t, func(tracer *elasticapm.Tracer) {
		tracer.Service.Environment = strings.Repeat("x", 1025)
	})
}

func TestValidateTransactionName(t *testing.T) {
	validatePayloads(t, func(tracer *elasticapm.Tracer) {
		tracer.StartTransaction(strings.Repeat("x", 1025), "type").End()
	})
}

func TestValidateTransactionType(t *testing.T) {
	validatePayloads(t, func(tracer *elasticapm.Tracer) {
		tracer.StartTransaction("name", strings.Repeat("x", 1025)).End()
	})
}

func TestValidateTransactionResult(t *testing.T) {
	validatePayloads(t, func(tracer *elasticapm.Tracer) {
		tx := tracer.StartTransaction("name", "type")
		tx.Result = strings.Repeat("x", 1025)
		tx.End()
	})
}

func TestValidateSpanName(t *testing.T) {
	validateTransaction(t, func(tx *elasticapm.Transaction) {
		tx.StartSpan(strings.Repeat("x", 1025), "type", nil).End()
	})
}

func TestValidateSpanType(t *testing.T) {
	validateTransaction(t, func(tx *elasticapm.Transaction) {
		tx.StartSpan("name", strings.Repeat("x", 1025), nil).End()
	})
}

func TestValidateContextUser(t *testing.T) {
	validateTransaction(t, func(tx *elasticapm.Transaction) {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		req.SetBasicAuth(strings.Repeat("x", 1025), "")
		tx.Context.SetHTTPRequest(req)
	})
}

func TestValidateContextCustom(t *testing.T) {
	t.Run("long_key", func(t *testing.T) {
		// NOTE(axw) this should probably fail, but does not. See:
		// https://github.com/elastic/apm-server/issues/910
		validateTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetCustom(strings.Repeat("x", 1025), "x")
		})
	})
	t.Run("reserved_key_chars", func(t *testing.T) {
		validateTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetCustom("x.y", "z")
		})
	})
}

func TestValidateContextTags(t *testing.T) {
	t.Run("long_key", func(t *testing.T) {
		// NOTE(axw) this should probably fail, but does not. See:
		// https://github.com/elastic/apm-server/issues/910
		validateTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetTag(strings.Repeat("x", 1025), "x")
		})
	})
	t.Run("long_value", func(t *testing.T) {
		validateTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetTag("x", strings.Repeat("x", 1025))
		})
	})
	t.Run("reserved_key_chars", func(t *testing.T) {
		validateTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetTag("x.y", "z")
		})
	})
}

func TestValidateRequestMethod(t *testing.T) {
	validateTransaction(t, func(tx *elasticapm.Transaction) {
		req, _ := http.NewRequest(strings.Repeat("x", 1025), "/", nil)
		tx.Context.SetHTTPRequest(req)
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
			validateTransaction(t, func(tx *elasticapm.Transaction) {
				req, _ := http.NewRequest("GET", test.url, nil)
				tx.Context.SetHTTPRequest(req)
			})
		})
	}
}

func TestValidateErrorException(t *testing.T) {
	t.Run("empty_message", func(t *testing.T) {
		validatePayloads(t, func(tracer *elasticapm.Tracer) {
			tracer.NewError(&testError{
				message: "",
			}).Send()
		})
	})
	t.Run("code", func(t *testing.T) {
		validatePayloads(t, func(tracer *elasticapm.Tracer) {
			tracer.NewError(&testError{
				message: "xyz",
				code:    strings.Repeat("x", 1025),
			}).Send()
		})
	})
	t.Run("type", func(t *testing.T) {
		validatePayloads(t, func(tracer *elasticapm.Tracer) {
			tracer.NewError(&testError{
				message: "xyz",
				type_:   strings.Repeat("x", 1025),
			}).Send()
		})
	})
}

func TestValidateErrorLog(t *testing.T) {
	tests := map[string]elasticapm.ErrorLogRecord{
		"empty_message": {
			Message: "",
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
			validatePayloads(t, func(tracer *elasticapm.Tracer) {
				tracer.NewErrorLog(record).Send()
			})
		})
	}
}

func validateTransaction(t *testing.T, f func(tx *elasticapm.Transaction)) {
	validatePayloads(t, func(tracer *elasticapm.Tracer) {
		tx := tracer.StartTransaction("name", "type")
		f(tx)
		tx.End()
	})
}

func validatePayloadMetadata(t *testing.T, f func(tracer *elasticapm.Tracer)) {
	validatePayloads(t, func(tracer *elasticapm.Tracer) {
		f(tracer)
		tracer.StartTransaction("name", "type").End()
	})
}

var (
	serverPkg         *build.Package
	serverPkgErr      error
	transactionSchema *jsonschema.Schema
	errorSchema       *jsonschema.Schema
)

func validatePayloads(t *testing.T, f func(tracer *elasticapm.Tracer)) {
	if serverPkg == nil && serverPkgErr == nil {
		serverPkg, serverPkgErr = build.Default.Import("github.com/elastic/apm-server", "", build.FindOnly)
	}
	if serverPkgErr != nil {
		t.Logf("couldn't find github.com/elastic/apm-server: %s", serverPkgErr)
		t.SkipNow()
	} else if transactionSchema == nil {
		var err error
		compiler := jsonschema.NewCompiler()
		specDir := path.Join(filepath.ToSlash(serverPkg.Dir), "docs/spec")
		transactionSchema, err = compiler.Compile("file://" + path.Join(specDir, "transactions/payload.json"))
		require.NoError(t, err)
		errorSchema, err = compiler.Compile("file://" + path.Join(specDir, "errors/payload.json"))
		require.NoError(t, err)
	}
	tracer, _ := elasticapm.NewTracer("tracer_testing", "")
	defer tracer.Close()
	tracer.Service.Name = "x"
	tracer.Service.Version = "x"
	tracer.Service.Environment = "x"
	tracer.Transport = &validatingTransport{
		t: t,
	}
	f(tracer)
	tracer.Flush(nil)
}

type validatingTransport struct {
	t *testing.T
	w fastjson.Writer
}

func (t *validatingTransport) SendTransactions(ctx context.Context, p *model.TransactionsPayload) error {
	t.validate(p, transactionSchema)
	return nil
}

func (t *validatingTransport) SendErrors(ctx context.Context, p *model.ErrorsPayload) error {
	t.validate(p, errorSchema)
	return nil
}

func (t *validatingTransport) validate(payload fastjson.Marshaler, schema *jsonschema.Schema) {
	t.w.Reset()
	payload.MarshalFastJSON(&t.w)
	err := schema.Validate(bytes.NewReader(t.w.Bytes()))
	require.NoError(t.t, err)
}

type testError struct {
	message string
	code    string
	type_   string
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
