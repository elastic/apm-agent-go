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

//go:build go1.11
// +build go1.11

package apmrestfulv3 // import "go.elastic.co/apm/module/apmrestfulv3"

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
