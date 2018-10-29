package model_test

import (
	"encoding/json"
	"testing"

	"go.elastic.co/fastjson"
)

func BenchmarkMarshalTransactionFastJSON(b *testing.B) {
	p := fakeTransaction()
	b.ResetTimer()

	var w fastjson.Writer
	for i := 0; i < b.N; i++ {
		p.MarshalFastJSON(&w)
		b.SetBytes(int64(w.Size()))
		w.Reset()
	}
}

func BenchmarkMarshalTransactionStdlib(b *testing.B) {
	p := fakeTransaction()
	b.ResetTimer()

	var cw countingWriter
	encoder := json.NewEncoder(&cw)
	for i := 0; i < b.N; i++ {
		encoder.Encode(p)
		b.SetBytes(int64(cw.n))
		cw.n = 0
	}
}

type countingWriter struct {
	n int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}
