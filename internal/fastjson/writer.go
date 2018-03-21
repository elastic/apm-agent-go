package fastjson

import (
	"strconv"
	"unicode/utf8"
)

type Writer struct {
	buf []byte
}

// Bytes returns the internal buffer. The result
// is invalidated when Reset is called.
func (w *Writer) Bytes() []byte {
	return w.buf
}

func (w *Writer) Reset() {
	w.buf = w.buf[:0]
}

func (w *Writer) RawByte(c byte) {
	w.buf = append(w.buf, c)
}

func (w *Writer) RawBytes(data []byte) {
	w.buf = append(w.buf, data...)
}

func (w *Writer) RawString(s string) {
	w.buf = append(w.buf, s...)
}

func (w *Writer) Uint64(n uint64) {
	w.buf = strconv.AppendUint(w.buf, uint64(n), 10)
}

func (w *Writer) Int64(n int64) {
	w.buf = strconv.AppendInt(w.buf, int64(n), 10)
}

func (w *Writer) Float32(n float32) {
	w.buf = strconv.AppendFloat(w.buf, float64(n), 'g', -1, 32)
}

func (w *Writer) Float64(n float64) {
	w.buf = strconv.AppendFloat(w.buf, float64(n), 'g', -1, 64)
}

func (w *Writer) Bool(v bool) {
	w.buf = strconv.AppendBool(w.buf, v)
}

const chars = "0123456789abcdef"

func isNotEscapedSingleChar(c byte, escapeHTML bool) bool {
	// Note: might make sense to use a table if there are more chars to escape. With 4 chars
	// it benchmarks the same.
	if escapeHTML {
		return c != '<' && c != '>' && c != '&' && c != '\\' && c != '"' && c >= 0x20 && c < utf8.RuneSelf
	} else {
		return c != '\\' && c != '"' && c >= 0x20 && c < utf8.RuneSelf
	}
}

func (w *Writer) String(s string) {
	w.RawByte('"')

	// Portions of the string that contain no escapes are appended as
	// byte slices.

	p := 0 // last non-escape symbol

	for i := 0; i < len(s); {
		c := s[i]

		if isNotEscapedSingleChar(c, true) {
			// single-width character, no escaping is required
			i++
			continue
		} else if c < utf8.RuneSelf {
			// single-with character, need to escape
			w.RawString(s[p:i])
			switch c {
			case '\t':
				w.RawString(`\t`)
			case '\r':
				w.RawString(`\r`)
			case '\n':
				w.RawString(`\n`)
			case '\\':
				w.RawString(`\\`)
			case '"':
				w.RawString(`\"`)
			default:
				w.RawString(`\u00`)
				w.RawByte(chars[c>>4])
				w.RawByte(chars[c&0xf])
			}

			i++
			p = i
			continue
		}

		// broken utf
		runeValue, runeWidth := utf8.DecodeRuneInString(s[i:])
		if runeValue == utf8.RuneError && runeWidth == 1 {
			w.RawString(s[p:i])
			w.RawString(`\ufffd`)
			i++
			p = i
			continue
		}

		// jsonp stuff - tab separator and line separator
		if runeValue == '\u2028' || runeValue == '\u2029' {
			w.RawString(s[p:i])
			w.RawString(`\u202`)
			w.RawByte(chars[runeValue&0xf])
			i += runeWidth
			p = i
			continue
		}
		i += runeWidth
	}
	w.RawString(s[p:])
	w.RawByte('"')
}
