// Copyright 2026 Elasticsearch B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific
// language governing permissions and limitations under the License.

package apm

import "testing"

// Regression for #1719: when the underlying bytes.Buffer is reused and
// grows past the configured cap, limitedBuffer.Write must not panic on
// the negative remaining-bytes math.
func TestLimitedBufferOverfullDoesNotPanic(t *testing.T) {
	b := &limitedBuffer{}
	// Push the underlying buffer past the cap by hand.
	for b.Len() <= stringLengthLimit*4 {
		_, _ = b.Buffer.Write(make([]byte, 1024))
	}
	if _, err := b.Write(make([]byte, 512)); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
}
