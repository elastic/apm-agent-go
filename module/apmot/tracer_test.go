package apmot_test

import (
	"net/url"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmot"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTransactionType(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	type test struct {
		Tag  opentracing.Tag
		Type string
	}
	tests := []test{
		{Tag: opentracing.Tag{Key: "component", Value: "foo"}, Type: "foo"},
		{Tag: opentracing.Tag{Key: "http.url", Value: "http://host/path"}, Type: "request"},
		{Tag: opentracing.Tag{Key: "foo", Value: "bar"}, Type: "unknown"}, // default
		{Tag: opentracing.Tag{Key: "type", Value: "baz"}, Type: "baz"},
	}
	for _, test := range tests {
		span := tracer.StartSpan("name", test.Tag)
		span.Finish()
	}

	apmtracer.Flush(nil)
	require.Len(t, recorder.Payloads(), 1)
	transactions := recorder.Payloads()[0].Transactions()
	require.Len(t, transactions, len(tests))
	for i, test := range tests {
		assert.Equal(t, test.Type, transactions[i].Type)
	}
}

func TestHTTPTransaction(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	span := tracer.StartSpan("name")
	ext.HTTPUrl.Set(span, "/foo?bar=baz")
	ext.HTTPMethod.Set(span, "POST")
	ext.HTTPStatusCode.Set(span, 404)
	span.Finish()

	apmtracer.Flush(nil)
	require.Len(t, recorder.Payloads(), 1)
	transactions := recorder.Payloads()[0].Transactions()
	require.Len(t, transactions, 1)
	transaction := transactions[0]
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)
	assert.Equal(t, &model.Request{
		Method:      "POST",
		HTTPVersion: "1.1",
		URL: model.URL{
			Protocol: "http",
			Path:     "/foo",
			Search:   "bar=baz",
		},
	}, transaction.Context.Request)
}

func TestSpanType(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	type test struct {
		Tag  opentracing.Tag
		Type string
	}
	tests := []test{
		{Tag: opentracing.Tag{Key: "component", Value: "foo"}, Type: "foo"},
		{Tag: opentracing.Tag{Key: "db.type", Value: "sql"}, Type: "db.sql.query"},
		{Tag: opentracing.Tag{Key: "http.url", Value: "http://testing.invalid:8000"}, Type: "ext.http"},
		{Tag: opentracing.Tag{Key: "foo", Value: "bar"}, Type: "unknown"}, // default
		{Tag: opentracing.Tag{Key: "type", Value: "baz"}, Type: "baz"},
	}

	txSpan := tracer.StartSpan("tx")
	for _, test := range tests {
		span := tracer.StartSpan("child", opentracing.ChildOf(txSpan.Context()), test.Tag)
		ext.SpanKindRPCClient.Set(span)
		span.Finish()
	}
	txSpan.Finish()

	apmtracer.Flush(nil)
	require.Len(t, recorder.Payloads(), 1)
	transactions := recorder.Payloads()[0].Transactions()
	require.Len(t, transactions, 1)
	require.Len(t, transactions[0].Spans, len(tests))
	for i, test := range tests {
		assert.Equal(t, test.Type, transactions[0].Spans[i].Type)
	}
}

func TestDBSpan(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	txSpan := tracer.StartSpan("tx")
	span := tracer.StartSpan("child", opentracing.ChildOf(txSpan.Context()))
	ext.DBInstance.Set(span, "test_db")
	ext.DBStatement.Set(span, "SELECT * FROM foo")
	ext.DBType.Set(span, "hbase")
	ext.DBUser.Set(span, "jean")
	span.Finish()
	txSpan.Finish()

	apmtracer.Flush(nil)
	require.Len(t, recorder.Payloads(), 1)
	transactions := recorder.Payloads()[0].Transactions()
	require.Len(t, transactions, 1)
	require.Len(t, transactions[0].Spans, 1)
	modelSpan := transactions[0].Spans[0]
	assert.Equal(t, "db.hbase.query", modelSpan.Type)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:  "test_db",
			Statement: "SELECT * FROM foo",
			Type:      "hbase",
			User:      "jean",
		},
	}, modelSpan.Context)
}

func TestHTTPSpan(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	clientURL := "https://root:hunter2@testing.invalid:8443/foo?bar:baz"
	url, err := url.Parse(clientURL)
	require.NoError(t, err)
	url.User = nil // user/password should be stripped off

	txSpan := tracer.StartSpan("tx")
	span := tracer.StartSpan("child", opentracing.ChildOf(txSpan.Context()))
	ext.HTTPMethod.Set(span, "GET")
	ext.HTTPUrl.Set(span, clientURL)
	span.Finish()
	txSpan.Finish()

	apmtracer.Flush(nil)
	require.Len(t, recorder.Payloads(), 1)
	transactions := recorder.Payloads()[0].Transactions()
	require.Len(t, transactions, 1)
	require.Len(t, transactions[0].Spans, 1)
	modelSpan := transactions[0].Spans[0]
	assert.Equal(t, "ext.http", modelSpan.Type)
	assert.Equal(t, &model.SpanContext{
		HTTP: &model.HTTPSpanContext{URL: url},
	}, modelSpan.Context)
}

func TestStartSpanRemoteParent(t *testing.T) {
	tracer1, apmtracer1, recorder1 := newTestTracer()
	defer apmtracer1.Close()
	tracer2, apmtracer2, recorder2 := newTestTracer()
	defer apmtracer1.Close()

	parentSpan := tracer1.StartSpan("parent")
	childSpan := tracer2.StartSpan("child", opentracing.ChildOf(parentSpan.Context()))
	childSpan.Finish()
	parentSpan.Finish()

	apmtracer1.Flush(nil)
	apmtracer2.Flush(nil)
	require.Len(t, recorder1.Payloads(), 1)
	require.Len(t, recorder2.Payloads(), 1)
}

func newTestTracer() (opentracing.Tracer, *elasticapm.Tracer, *transporttest.RecorderTransport) {
	apmtracer, recorder := transporttest.NewRecorderTracer()
	tracer := apmot.New(apmot.WithTracer(apmtracer))
	return tracer, apmtracer, recorder
}
