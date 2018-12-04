package apmot_test

import (
	"context"
	"net/url"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmot"
	"go.elastic.co/apm/transport/transporttest"
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
		{Tag: opentracing.Tag{Key: "foo", Value: "bar"}, Type: "custom"}, // default
		{Tag: opentracing.Tag{Key: "type", Value: "baz"}, Type: "baz"},
	}
	for _, test := range tests {
		span := tracer.StartSpan("name", test.Tag)
		span.Finish()
	}

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	transactions := payloads.Transactions
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
	payloads := recorder.Payloads()
	require.Len(t, payloads.Transactions, 1)
	transaction := payloads.Transactions[0]
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
		Tag     opentracing.Tag
		Type    string
		Subtype string
	}
	tests := []test{
		{Tag: opentracing.Tag{Key: "component", Value: "foo"}, Type: "custom", Subtype: "foo"},
		{Tag: opentracing.Tag{Key: "db.type", Value: "sql"}, Type: "db", Subtype: "sql"},
		{Tag: opentracing.Tag{Key: "http.url", Value: "http://testing.invalid:8000"}, Type: "external", Subtype: "http"},
		{Tag: opentracing.Tag{Key: "foo", Value: "bar"}, Type: "custom"}, // default
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
	payloads := recorder.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, len(tests))
	for i, test := range tests {
		assert.Equal(t, test.Type, payloads.Spans[i].Type)
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
	payloads := recorder.Payloads()
	require.Len(t, payloads.Spans, 1)
	modelSpan := payloads.Spans[0]
	assert.Equal(t, "db", modelSpan.Type)
	assert.Equal(t, "hbase", modelSpan.Subtype)
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
	payloads := recorder.Payloads()
	require.Len(t, payloads.Spans, 1)
	modelSpan := payloads.Spans[0]
	assert.Equal(t, "external", modelSpan.Type)
	assert.Equal(t, "http", modelSpan.Subtype)
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
	require.Len(t, recorder1.Payloads().Transactions, 1)
	require.Len(t, recorder2.Payloads().Transactions, 1)
}

func TestStartSpanParentFinished(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	parentSpan := tracer.StartSpan("parent")
	parentSpan.Finish()

	childSpan := tracer.StartSpan("child", opentracing.ChildOf(parentSpan.Context()))
	childSpan.Finish()

	grandChildSpan := tracer.StartSpan("grandchild", opentracing.ChildOf(childSpan.Context()))
	grandChildSpan.Finish()

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	assert.Len(t, payloads.Spans, 2)

	tx := payloads.Transactions[0]
	assert.Equal(t, tx.ID, payloads.Spans[0].ParentID)
	assert.Equal(t, payloads.Spans[0].ID, payloads.Spans[1].ParentID)
	for _, span := range payloads.Spans {
		assert.Equal(t, tx.ID, span.TransactionID)
		assert.Equal(t, tx.TraceID, span.TraceID)
	}
}

func TestCustomTags(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	outer := tracer.StartSpan("name", opentracing.Tag{Key: "foo", Value: "bar"})
	inner := tracer.StartSpan("name", opentracing.Tag{Key: "baz", Value: "qux"}, opentracing.ChildOf(outer.Context()))
	inner.Finish()
	outer.Finish()

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	assert.Equal(t, model.StringMap{{Key: "foo", Value: "bar"}}, payloads.Transactions[0].Context.Tags)
	assert.Equal(t, model.StringMap{{Key: "baz", Value: "qux"}}, payloads.Spans[0].Context.Tags)
}

func TestStartSpanFromContextMixed(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()
	opentracing.SetGlobalTracer(tracer)

	tx := apmtracer.StartTransaction("tx", "unknown")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	apmSpan1, ctx := apm.StartSpan(ctx, "apm1", "apm")
	otSpan1, ctx := opentracing.StartSpanFromContext(ctx, "ot1")
	apmSpan2, ctx := apm.StartSpan(ctx, "apm2", "apm")
	otSpan2, ctx := opentracing.StartSpanFromContext(ctx, "ot2")
	otSpan3, ctx := opentracing.StartSpanFromContext(ctx, "ot3")
	otSpan3.Finish()
	otSpan2.Finish()
	apmSpan2.End()
	otSpan1.Finish()
	apmSpan1.End()
	tx.End()

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 5)

	assert.Equal(t, "ot3", payloads.Spans[0].Name)
	assert.Equal(t, "ot2", payloads.Spans[1].Name)
	assert.Equal(t, "apm2", payloads.Spans[2].Name)
	assert.Equal(t, "ot1", payloads.Spans[3].Name)
	assert.Equal(t, "apm1", payloads.Spans[4].Name)
	assert.Equal(t, payloads.Spans[4].ID, payloads.Spans[3].ParentID)
	assert.Equal(t, payloads.Spans[3].ID, payloads.Spans[2].ParentID)
	assert.Equal(t, payloads.Spans[2].ID, payloads.Spans[1].ParentID)
	assert.Equal(t, payloads.Spans[1].ID, payloads.Spans[0].ParentID)
}

func newTestTracer() (opentracing.Tracer, *apm.Tracer, *transporttest.RecorderTransport) {
	apmtracer, recorder := transporttest.NewRecorderTracer()
	tracer := apmot.New(apmot.WithTracer(apmtracer))
	return tracer, apmtracer, recorder
}
