package model

import (
	"encoding/json"
	"errors"
)

const (
	// YYYY-MM-DDTHH:mm:ss.sssZ
	dateTimeFormat = "2006-01-02T15:04:05.999Z"
)

// MarshalJSON returns the JSON encoding of t.
func (t *Transaction) MarshalJSON() ([]byte, error) {
	// Wrap the Transaction type so we can format the timestamp and
	// duration fields according to the JSON Schema.
	type TransactionInternal Transaction
	var ti = struct {
		*TransactionInternal
		Timestamp string  `json:"timestamp"`
		Duration  float64 `json:"duration"`
	}{
		(*TransactionInternal)(t),
		t.Timestamp.UTC().Format(dateTimeFormat),
		t.Duration.Seconds() * 1000,
	}
	return json.Marshal(ti)
}

// MarshalJSON returns the JSON encoding of s.
func (s *Span) MarshalJSON() ([]byte, error) {
	// Wrap the Span type so we can format the start and
	// duration fields according to the JSON Schema.
	type SpanInternal Span
	var si = struct {
		*SpanInternal
		Start    float64 `json:"start"`
		Duration float64 `json:"duration"`
	}{
		(*SpanInternal)(s),
		s.Start.Seconds() * 1000,
		s.Duration.Seconds() * 1000,
	}
	return json.Marshal(si)
}

// MarshalJSON returns the JSON encoding of r.
func (r *Request) MarshalJSON() ([]byte, error) {
	var cookies map[string]interface{}
	if len(r.Cookies) != 0 {
		cookies = make(map[string]interface{}, len(r.Cookies))
		for _, c := range r.Cookies {
			cookies[c.Name] = c.Value
		}
	}

	// Wrap the Request type so we can format the URL
	// and cookies fields according to the JSON Schema.
	type RequestInternal Request
	var ri = struct {
		*RequestInternal
		Cookies map[string]interface{} `json:"cookies,omitempty"`
	}{
		(*RequestInternal)(r), cookies,
	}
	return json.Marshal(ri)
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

// MarshalJSON returns the JSON encoding of e.
func (e *Error) MarshalJSON() ([]byte, error) {
	// Wrap the Error type so we can format the timestamp and
	// duration fields according to the JSON Schema.
	type ErrorInternal Error
	type TransactionInternal struct {
		ID string `json:"id"`
	}
	var transaction *TransactionInternal
	if e.TransactionID != "" {
		transaction = &TransactionInternal{ID: e.TransactionID}
	}
	var ei = struct {
		*ErrorInternal
		Timestamp   string               `json:"timestamp"`
		Transaction *TransactionInternal `json:"transaction,omitempty"`
	}{
		(*ErrorInternal)(e),
		e.Timestamp.UTC().Format(dateTimeFormat),
		transaction,
	}
	return json.Marshal(ei)
}
