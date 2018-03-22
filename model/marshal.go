package model

import (
	"time"

	"github.com/elastic/apm-agent-go/internal/fastjson"
)

//go:generate go run ../internal/fastjson/generate.go -f -o marshal_fastjson.go .

const (
	// YYYY-MM-DDTHH:mm:ss.sssZ
	dateTimeFormat = "2006-01-02T15:04:05.999Z"
)

// FormatTime formats the time.Time, in UTC, in the format expected
// for Transaction.Timestamp.
func FormatTime(t time.Time) string {
	return t.UTC().Format(dateTimeFormat)
}

func (c Cookies) isZero() bool {
	return len(c) == 0
}

func (c Cookies) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	first := true
outer:
	for i := len(c) - 1; i >= 0; i-- {
		for j := i + 1; j < len(c); j++ {
			if c[i].Name == c[j].Name {
				continue outer
			}
		}
		if first {
			first = false
		} else {
			w.RawByte(',')
		}
		w.String(c[i].Name)
		w.RawByte(':')
		w.String(c[i].Value)
	}
	w.RawByte('}')
}

// isZero is used by fastjson to implement omitempty.
func (t *ErrorTransaction) isZero() bool {
	return t.ID == ""
}

// MarshalFastJSON writes the JSON representation of c to w.
func (c *ExceptionCode) MarshalFastJSON(w *fastjson.Writer) {
	if c.String != "" {
		w.String(c.String)
		return
	}
	w.Float64(c.Number)
}

// isZero is used by fastjson to implement omitempty.
func (c *ExceptionCode) isZero() bool {
	return c.String == "" && c.Number == 0
}

// MarshalFastJSON writes the JSON representation of b to w.
func (b *RequestBody) MarshalFastJSON(w *fastjson.Writer) {
	if b.Form != nil {
		w.RawByte('{')
		first := true
		for k, v := range b.Form {
			if first {
				first = false
			} else {
				w.RawByte(',')
			}
			w.String(k)
			w.RawByte(':')
			if len(v) == 1 {
				// Just one item, add the item directly.
				w.String(v[0])
			} else {
				// Zero or multiple items, include them all.
				w.RawByte('[')
				first := true
				for _, v := range v {
					if first {
						first = false
					} else {
						w.RawByte(',')
					}
					w.String(v)
				}
				w.RawByte(']')
			}
		}
		w.RawByte('}')
	} else {
		w.String(b.Raw)
	}
}
