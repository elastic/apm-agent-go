package sqlscanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanEOF(t *testing.T) {
	s := NewScanner("  ")
	assert.False(t, s.Scan())
}

func TestScanKeyword(t *testing.T) {
	s := NewScanner("INSERT or rEpLaCe")
	assertScan(t, s, INSERT, "INSERT")
	assertScan(t, s, OR, "or")
	assertScan(t, s, REPLACE, "rEpLaCe")
}

func TestScanQualifiedTable(t *testing.T) {
	s := NewScanner("schema.Abc_123")
	assertScan(t, s, IDENT, "schema")
	assertScan(t, s, PERIOD, ".")
	assertScan(t, s, IDENT, "Abc_123")
}

func TestScanVariable(t *testing.T) {
	s := NewScanner("$123")
	assertScan(t, s, OTHER, "$123")
}

func TestScanIdent(t *testing.T) {
	s := NewScanner("_foo foo$")
	assertScan(t, s, IDENT, "_foo")
	assertScan(t, s, IDENT, "foo$")
}

func TestScanQuotedIdent(t *testing.T) {
	s := NewScanner("`SELECT` \"SELECT \"\"\" [SELECT '']")
	assertScan(t, s, IDENT, "SELECT")
	assertScan(t, s, IDENT, `SELECT ""`)
	assertScan(t, s, IDENT, "SELECT ''")
}

func TestScanComment(t *testing.T) {
	s := NewScanner("/* /*nested*/ */ -- SELECT /*")
	assertScan(t, s, COMMENT, "/* /*nested*/ */")
	assertScan(t, s, COMMENT, "-- SELECT /*")
}

func TestScanString(t *testing.T) {
	s := NewScanner("'abc '' def\\''")
	assertScan(t, s, STRING, "'abc '' def\\''")
}

func TestScanDollarQuotedString(t *testing.T) {
	s := NewScanner("$$f$o$o$$ $$ $$ $foo$'`$$$$\"$foo$ $foo $bar")
	assertScan(t, s, STRING, "$$f$o$o$$")
	assertScan(t, s, STRING, "$$ $$")
	assertScan(t, s, STRING, "$foo$'`$$$$\"$foo$")
	assertScan(t, s, OTHER, "$foo")
	assertScan(t, s, OTHER, "$bar")

	// Unterminated dollar-quoted string stops tokenizing
	// at the first whitespace, under the assumption that
	// the input is valid and we've interpreted it wrongly.
	s = NewScanner("$foo$ banana $")
	assertScan(t, s, OTHER, "$foo$")
	assertScan(t, s, IDENT, "banana")
	assertScan(t, s, OTHER, "$")
}

func TestScanNumber(t *testing.T) {
	s := NewScanner("123 123.456 123E45 123e+45 123e-45")
	assertScan(t, s, NUMBER, "123")
	assertScan(t, s, NUMBER, "123.456")
	assertScan(t, s, NUMBER, "123E45")
	assertScan(t, s, NUMBER, "123e+45")
	assertScan(t, s, NUMBER, "123e-45")
}

func assertScan(t *testing.T, s *Scanner, tok Token, text string) {
	assert.True(t, s.Scan())
	assert.Equal(t, tok, s.Token())
	assert.Equal(t, text, s.Text())
}
