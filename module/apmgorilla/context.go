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

package apmgorilla

import (
	"bytes"
	"strings"
)

// massageTemplate removes the regexp patterns from template variables.
func massageTemplate(tpl string) string {
	braces := braceIndices(tpl)
	if len(braces) == 0 {
		return tpl
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(tpl)))
	for i := 0; i < len(tpl); {
		var j int
		if i < braces[0] {
			j = braces[0]
			buf.WriteString(tpl[i:j])
		} else {
			j = braces[1]
			field := tpl[i:j]
			if colon := strings.IndexRune(field, ':'); colon >= 0 {
				buf.WriteString(field[:colon])
				buf.WriteRune('}')
			} else {
				buf.WriteString(field)
			}
			braces = braces[2:]
			if len(braces) == 0 {
				buf.WriteString(tpl[j:])
				break
			}
		}
		i = j
	}
	return buf.String()
}

// Copied/adapted from gorilla/mux (see NOTICE). The original version
// checks that the braces are matched up correctly; we assume they are,
// as otherwise the path wouldn't have been registered correctly.
func braceIndices(s string) []int {
	var level, idx int
	var idxs []int
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			if level++; level == 1 {
				idx = i
			}
		case '}':
			if level--; level == 0 {
				idxs = append(idxs, idx, i+1)
			}
		}
	}
	return idxs
}
