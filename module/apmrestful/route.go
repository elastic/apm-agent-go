package apmrestful

import (
	"bytes"
	"strings"
)

// massageRoutePath removes the regexp patterns from route variables.
func massageRoutePath(route string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(route)))
	end := 0
	for end < len(route) {
		var token string
		i := strings.IndexRune(route[end:], '/')
		if i == -1 {
			token = route[end:]
			end = len(route)
		} else {
			token = route[end : end+i+1]
			end += i + 1
		}
		if strings.HasPrefix(token, "{") {
			colon := strings.IndexRune(token, ':')
			if colon != -1 {
				buf.WriteString(token[:colon])
				rbracket := strings.LastIndexByte(token[colon:], '}')
				if rbracket != -1 {
					buf.WriteString(token[colon+rbracket:])
				}
			} else {
				buf.WriteString(token)
			}
		} else {
			buf.WriteString(token)
		}
	}
	return buf.String()
}
