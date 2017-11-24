package model_test

import (
	"encoding/json"
	"testing"
)

func BenchmarkMarshalTransactionStdlib(b *testing.B) {
	t := fakeTransaction()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(t); err != nil {
			b.Fatalf("encoding/json.Marshal failed: %v", err)
			return
		}
	}
}
