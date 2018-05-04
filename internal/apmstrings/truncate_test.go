package apmstrings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/internal/apmstrings"
)

func TestTruncate(t *testing.T) {
	const limit = 2
	test := func(name, in, expect string) {
		t.Run(name, func(t *testing.T) {
			out := apmstrings.Truncate(in, limit)
			assert.Equal(t, out, expect)
		})
	}
	test("empty", "", "")
	test("limit_ascii", "xx", "xx")
	test("limit_multibyte", "世界", "世界")
	test("truncate_ascii", "xxx", "xx")
	test("truncate_multibyte", "世界世", "世界")
}
