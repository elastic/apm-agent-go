package model

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/pkg/errors"

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

// MarshalFastJSON writes the JSON representation of c to w.
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

// UnmarshalJSON unmarshals the JSON data into c.
func (c *Cookies) UnmarshalJSON(data []byte) error {
	m := make(map[string]string)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*c = make([]*http.Cookie, 0, len(m))
	for k, v := range m {
		*c = append(*c, &http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	sort.Slice(*c, func(i, j int) bool {
		return (*c)[i].Name < (*c)[j].Name
	})
	return nil
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

// UnmarshalJSON unmarshals the JSON data into c.
func (c *ExceptionCode) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		c.String = v
	case float64:
		c.Number = v
	default:
		return errors.Errorf("expected string or number, got %T", v)
	}
	return nil
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

// UnmarshalJSON unmarshals the JSON data into b.
func (b *RequestBody) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		b.Raw = v
		return nil
	case map[string]interface{}:
		for k, v := range v {
			switch v := v.(type) {
			case string:
				b.Form.Set(k, v)
			case []interface{}:
				for _, v := range v {
					switch v := v.(type) {
					case string:
						b.Form.Add(k, v)
					default:
						return errors.Errorf("expected string, got %T", v)
					}
				}
			default:
				return errors.Errorf("expected string or []string, got %T", v)
			}
		}
	default:
		return errors.Errorf("expected string or map, got %T", v)
	}
	return nil
}
