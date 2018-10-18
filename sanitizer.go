package apm

import (
	"bytes"

	"go.elastic.co/apm/internal/wildcard"
	"go.elastic.co/apm/model"
)

const redacted = "[REDACTED]"

// sanitizeRequest sanitizes HTTP request data, redacting
// the values of cookies and forms whose corresponding keys
// match any of the given wildcard patterns.
func sanitizeRequest(r *model.Request, matchers wildcard.Matchers) {
	var anyCookiesRedacted bool
	for _, c := range r.Cookies {
		if !matchers.MatchAny(c.Name) {
			continue
		}
		c.Value = redacted
		anyCookiesRedacted = true
	}
	if anyCookiesRedacted && r.Headers != nil {
		var b bytes.Buffer
		for i, c := range r.Cookies {
			if i != 0 {
				b.WriteRune(';')
			}
			b.WriteString(c.String())
		}
		r.Headers.Cookie = b.String()
	}
	if r.Body != nil && r.Body.Form != nil {
		for key, values := range r.Body.Form {
			if !matchers.MatchAny(key) {
				continue
			}
			for i := range values {
				values[i] = redacted
			}
		}
	}
}
