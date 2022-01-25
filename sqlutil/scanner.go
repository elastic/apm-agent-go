// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sqlutil // import "go.elastic.co/apm/v2/sqlutil"

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scanner is the struct used to generate SQL
// tokens for the parser.
type Scanner struct {
	input string
	start int // text start pos in bytes
	end   int // text end pos in bytes
	pos   int // read pos in bytes
	tok   Token
}

// NewScanner creates a new Scanner for sql.
func NewScanner(sql string) *Scanner {
	return &Scanner{input: sql}
}

// Token returns the most recently scanned token.
func (s *Scanner) Token() Token {
	return s.tok
}

// Text returns the portion of the string that relates to
// the most recently scanned token.
func (s *Scanner) Text() string {
	return s.input[s.start:s.end]
}

// Scan scans for the next token and returns true if one was
// found, false if the end of the input stream was reached.
// When Scan returns true, the token type can be obtained by
// calling the Token() method, and the text can be obtained
// by calling the Text() method.
func (s *Scanner) Scan() bool {
	s.tok = s.scan()
	return s.tok != eof
}

func (s *Scanner) scan() Token {
	r, ok := s.next()
	if !ok {
		return eof
	}
	for unicode.IsSpace(r) {
		if r, ok = s.next(); !ok {
			return eof
		}
	}
	s.start = s.pos - utf8.RuneLen(r)

	if r == '_' || unicode.IsLetter(r) {
		return s.scanKeywordOrIdentifier(r != '_')
	} else if unicode.IsDigit(r) {
		return s.scanNumericLiteral()
	}

	switch r {
	case '\'':
		// Standard string literal.
		return s.scanStringLiteral()
	case '"':
		// Standard double-quoted identifier.
		//
		// NOTE(axw) MySQL will treat " as a
		// string literal delimiter by default,
		// but we assume standard SQL and treat
		// it as a identifier delimiter.
		return s.scanQuotedIdentifier('"')
	case '[':
		// T-SQL bracket-quoted identifier.
		return s.scanQuotedIdentifier(']')
	case '`':
		// MySQL-style backtick-quoted identifier.
		return s.scanQuotedIdentifier('`')
	case '(':
		return LPAREN
	case ')':
		return RPAREN
	case '-':
		if next, ok := s.peek(); ok && next == '-' {
			// -- comment
			s.next()
			return s.scanSimpleComment()
		}
		return OTHER
	case '/':
		if next, ok := s.peek(); ok {
			switch next {
			case '*':
				// /* comment */
				s.next()
				return s.scanBracketedComment()
			case '/':
				// // comment
				s.next()
				return s.scanSimpleComment()
			}
		}
		return OTHER
	case '.':
		return PERIOD
	case '$':
		next, ok := s.peek()
		if !ok {
			break
		}
		if unicode.IsDigit(next) {
			// This is a variable like "$1".
			for {
				if next, ok := s.peek(); !ok || !unicode.IsDigit(next) {
					break
				}
				s.next()
			}
			return OTHER
		} else if next == '$' || next == '_' || unicode.IsLetter(next) {
			// PostgreSQL supports dollar-quoted string literal syntax,
			// like $foo$...$foo$. The tag (foo in this case) is optional,
			// and if present follows identifier rules.
			for {
				r, ok := s.next()
				if !ok {
					// Unknown token starting with $ until
					// EOF, just ignore it.
					return OTHER
				}
				switch {
				case r == '$':
					// This marks the end of the initial $foo$.
					tag := s.Text()
					if i := strings.Index(s.input[s.pos:], tag); i >= 0 {
						s.end += i + len(tag)
						s.pos += i + len(tag)
						return STRING
					}
				case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
					// Identifier rune, consume.
				case unicode.IsSpace(r):
					// Unknown token starting with $,
					// consume runes until space.
					s.end -= utf8.RuneLen(r)
					return OTHER
				}
			}
		}
		return OTHER
	}
	return OTHER
}

func (s *Scanner) scanKeywordOrIdentifier(maybeKeyword bool) Token {
loop:
	for {
		r, ok := s.peek()
		if !ok {
			break loop
		}
		switch {
		case unicode.IsLetter(r):
		case unicode.IsDigit(r) || r == '_' || r == '$':
			maybeKeyword = false
		default:
			break loop
		}
		s.next()
	}
	if !maybeKeyword {
		return IDENT
	}
	text := s.Text()
	if len(text) >= len(keywords) {
		return IDENT
	}
	for _, token := range keywords[len(text)] {
		if strings.EqualFold(text, token.String()) {
			return token
		}
	}
	return IDENT
}

func (s *Scanner) scanQuotedIdentifier(delim rune) Token {
loop:
	for {
		r, ok := s.next()
		if !ok {
			return eof
		}
		if r == delim {
			if delim == '"' {
				if r, ok := s.peek(); ok && r == delim {
					// Skip escaped double quotes,
					// e.g. "He said ""great""".
					s.next()
					continue loop
				}
			}
			break
		}
	}
	// Remove quotes from identifier.
	s.start++
	s.end--
	return IDENT
}

func (s *Scanner) scanNumericLiteral() Token {
	var havePeriod bool
	var haveExponent bool
	for {
		r, ok := s.peek()
		if !ok {
			return NUMBER
		}
		if unicode.IsDigit(r) {
			s.next()
			continue
		}
		switch r {
		case '.':
			if havePeriod {
				return NUMBER
			}
			s.next()
			havePeriod = true
		case 'e', 'E':
			if haveExponent {
				return NUMBER
			}
			s.next()
			haveExponent = true
			if r, ok := s.peek(); ok && (r == '+' || r == '-') {
				s.next()
			}
		default:
			return NUMBER
		}
	}
}

func (s *Scanner) scanStringLiteral() Token {
	const delim = '\''
	for {
		r, ok := s.next()
		if !ok {
			return eof
		}
		if r == '\\' {
			// Skip escaped character, e.g. 'what\'s up?'
			s.next()
			continue
		}
		if r != delim {
			continue
		}
		if r, ok := s.peek(); !ok || r != delim {
			return STRING
		}
		// Two ' characters next to each other
		// are collapsed in a string literal,
		// rather than escaping the string. We
		// don't care about string values, so
		// we don't collapse.
		s.next()
	}
}

func (s *Scanner) scanSimpleComment() Token {
	for {
		if r, ok := s.next(); !ok || r == '\n' {
			return COMMENT
		}
	}
}

func (s *Scanner) scanBracketedComment() Token {
	nesting := 1
	for {
		r, ok := s.next()
		if !ok {
			return eof
		}
		switch r {
		case '/':
			r, ok := s.peek()
			if ok && r == '*' {
				s.next()
				nesting++
			}
		case '*':
			r, ok := s.peek()
			if ok && r == '/' {
				s.next()
				nesting--
				if nesting == 0 {
					return COMMENT
				}
			}
		}
	}
}

// next returns the next rune if there is one, and advances
// the scanner position, or returns utf8.RuneError if there
// is no valid next rune. The bool result indicates whether
// a valid rune is returned.
func (s *Scanner) next() (rune, bool) {
	r, rlen := s.peekLen()
	if r != utf8.RuneError {
		s.pos += rlen
		s.end = s.pos
		return r, true
	}
	return r, false
}

// peek returns the next rune if there is one, or
// utf8.RuneError if not. The bool result indicates
// whether a valid rune is returned.
func (s *Scanner) peek() (rune, bool) {
	r, _ := s.peekLen()
	if r == utf8.RuneError {
		return utf8.RuneError, false
	}
	return r, true
}

// peekLen returns the next rune (if there is one)
// and its length. If there is no next valid rune,
// utf8.RuneError and a length of -1 are returned.
func (s *Scanner) peekLen() (rune, int) {
	if s.pos >= len(s.input) {
		return utf8.RuneError, -1
	}
	return utf8.DecodeRuneInString(s.input[s.pos:])
}
