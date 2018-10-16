package wildcard

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWildcardStartsWith(t *testing.T) {
	mS := NewMatcher("foo*", CaseSensitive)
	mI := NewMatcher("foo*", CaseInsensitive)
	for _, m := range []*Matcher{mS, mI} {
		assert.True(t, m.Match("foo"))
		assert.True(t, m.Match("foobar"))
		assert.False(t, m.Match("bar"))
		assert.False(t, m.Match("barfoo"))
		assert.False(t, m.Match(""))
	}
	assert.True(t, mI.Match("FoO"))
	assert.False(t, mS.Match("FoO"))
}

func TestWildcardEndsWith(t *testing.T) {
	mS := NewMatcher("*foo", CaseSensitive)
	mI := NewMatcher("*foo", CaseInsensitive)
	for _, m := range []*Matcher{mS, mI} {
		assert.True(t, m.Match("foo"))
		assert.True(t, m.Match("barfoo"))
		assert.True(t, m.Match("\xed\xbf\xbf\x80foo"))
		assert.False(t, m.Match("foobar"))
		assert.False(t, m.Match("bar"))
		assert.False(t, m.Match(""))
	}
	assert.True(t, mI.Match("BaRFoO"))
	assert.False(t, mS.Match("BaRFoO"))
}

func TestWildcardEqual(t *testing.T) {
	mS := NewMatcher("foo", CaseSensitive)
	mI := NewMatcher("foo", CaseInsensitive)
	for _, m := range []*Matcher{mS, mI} {
		assert.True(t, m.Match("foo"))
		assert.False(t, m.Match("foobar"))
		assert.False(t, m.Match("bar"))
		assert.False(t, m.Match("barfoo"))
		assert.False(t, m.Match(""))
	}
	assert.True(t, mI.Match("FoO"))
	assert.False(t, mS.Match("FoO"))
}

func TestWildcardAll(t *testing.T) {
	mS := NewMatcher("*", CaseSensitive)
	mI := NewMatcher("*", CaseInsensitive)
	for _, m := range []*Matcher{mS, mI} {
		assert.True(t, m.Match(""))
		assert.True(t, m.Match("x"))
	}
}

func TestWildcardEmptyPattern(t *testing.T) {
	mS := NewMatcher("", CaseSensitive)
	mI := NewMatcher("", CaseInsensitive)
	for _, m := range []*Matcher{mS, mI} {
		assert.True(t, m.Match(""))
		assert.False(t, m.Match("x"))
	}
}

func TestWildcardMultiples(t *testing.T) {
	mS := NewMatcher("a*b*c", CaseSensitive)
	mI := NewMatcher("a*b*c", CaseInsensitive)
	for _, m := range []*Matcher{mS, mI} {
		assert.True(t, m.Match("abc"))
		assert.True(t, m.Match("abbc"))
		assert.True(t, m.Match("aabc"))
		assert.True(t, m.Match("abcc"))
		assert.False(t, m.Match("ab"))
		assert.False(t, m.Match("bc"))
		assert.False(t, m.Match("ac"))
		assert.False(t, m.Match("_abc_"))
		assert.False(t, m.Match(""))
	}
}

func TestWildcardSandwich(t *testing.T) {
	mS := NewMatcher("a*c", CaseSensitive)
	mI := NewMatcher("a*c", CaseInsensitive)
	for _, m := range []*Matcher{mS, mI} {
		assert.True(t, m.Match("abc"))
		assert.True(t, m.Match("abbc"))
		assert.True(t, m.Match("aabc"))
		assert.True(t, m.Match("abcc"))
		assert.True(t, m.Match("ac"))
		assert.False(t, m.Match("ab"))
		assert.False(t, m.Match("bc"))
		assert.False(t, m.Match("_abc_"))
		assert.False(t, m.Match(""))
	}
}

var benchmarkPatterns = []string{
	"password",
	"passwd",
	"pwd",
	"secret",
	"*key",
	"*token",
	"*session*",
	"*credit*",
	"*card*",
}

func BenchmarkWildcardMatcher(b *testing.B) {
	matchers := make(Matchers, len(benchmarkPatterns))
	for i, p := range benchmarkPatterns {
		matchers[i] = NewMatcher(p, CaseInsensitive)
	}
	b.ResetTimer()
	benchmarkMatch(b, matchers.MatchAny)
}

func BenchmarkRegexpMatcher(b *testing.B) {
	patterns := make([]string, len(benchmarkPatterns))
	for i, p := range benchmarkPatterns {
		patterns[i] = strings.Replace(p, "*", ".*", -1)
	}
	re := regexp.MustCompile(fmt.Sprintf("(?i:%s)", strings.Join(patterns, "|")))
	b.ResetTimer()
	benchmarkMatch(b, re.MatchString)
}

func benchmarkMatch(b *testing.B, match func(s string) bool) {
	var bytes int64
	test := func(s string, expect bool) {
		if match(s) != expect {
			panic("nope")
		}
		bytes += int64(len(s))
	}
	for i := 0; i < b.N; i++ {
		test("foo", false)
		test("session", true)
		test("passwork", false)
		test("pwd", true)
		test("credit_card", true)
		test("zing", false)
		test("boop", false)

		b.SetBytes(bytes)
		bytes = 0
	}
}
