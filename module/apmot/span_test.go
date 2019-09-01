package apmot

import (
	"go.elastic.co/apm"
	"testing"
)

func BenchmarkSetSpanContext(b *testing.B) {
	otSpan := &otSpan{
		span: &apm.Span{
			SpanData: &apm.SpanData{},
		},
		tags: map[string]interface{}{
			"component":    "myComponent",
			"db.instance":  "myDbInstance",
			"db.statement": "myStatement",
			"db.type":      "myDbType",
			"db.user":      "myUser",
			"http.url":     "myHttpUrl",
			"http.method":  "myHttpMethod",
			"type":         "myType",
			"custom1":      "myCustom1",
			"custom2":      "myCustom2",
		},
	}
	for n := 0; n < b.N; n++ {
		otSpan.setSpanContext()
	}
}

func BenchmarkSetTransactionContext(b *testing.B) {
	otSpan := &otSpan{
		ctx: spanContext{
			tx: &apm.Transaction{
				TransactionData: &apm.TransactionData{
					Context: apm.Context{},
				},
			},
		},
		tags: map[string]interface{}{
			"component":        "myComponent",
			"http.method":      "myHttpMethod",
			"http.status_code": 200,
			"http.url":         "myHttpUrl",
			"error":            false,
			"type":             "myType",
			"result":           "myResult",
			"user.id":          "myUserId",
			"user.email":       "myUserEmail",
			"user.username":    "myUserUserName",
			"custom1":          "myCustom1",
			"custom2":          "myCustom2",
		},
	}
	for n := 0; n < b.N; n++ {
		otSpan.setTransactionContext()
	}
}
