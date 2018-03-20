package model

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	// YYYY-MM-DDTHH:mm:ss.sssZ
	dateTimeFormat = "2006-01-02T15:04:05.999Z"
)

// FormatTime formats the time.Time, in UTC, in the format expected
// for Transaction.Timestamp.
func FormatTime(t time.Time) string {
	return t.UTC().Format(dateTimeFormat)
}

// MarshalJSON returns the JSON encoding of id.
func (id ErrorTransactionID) MarshalJSON() ([]byte, error) {
	out := struct {
		ID string `json:"id"`
	}{string(id)}
	return json.Marshal(out)
}

// MarshalJSON returns the JSON encoding of b.
func (b *RequestBody) MarshalJSON() ([]byte, error) {
	if b.Form != nil {
		if b.Raw != "" {
			return nil, errors.New("only one of Form and Raw may be set in Request.Body")
		}
		out := make(map[string]interface{})
		for k, v := range b.Form {
			if len(v) == 1 {
				// Just one item, add the item directly.
				out[k] = v[0]
			} else {
				// Zero or multiple items, include them all.
				out[k] = v
			}
		}
	}
	return json.Marshal(b.Raw)
}
