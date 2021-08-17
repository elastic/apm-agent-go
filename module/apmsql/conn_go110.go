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

//go:build go1.10
// +build go1.10

package apmsql // import "go.elastic.co/apm/module/apmsql"

import (
	"context"
	"database/sql/driver"
)

// Support for Conn interfaces introduced in Go 1.10 and later.
type connGo110 struct {
	sessionResetter driver.SessionResetter
}

func (c *connGo110) init(in driver.Conn) {
	c.sessionResetter, _ = in.(driver.SessionResetter)
}

func (c *connGo110) ResetSession(ctx context.Context) error {
	if c.sessionResetter != nil {
		return c.sessionResetter.ResetSession(ctx)
	}
	return nil
}
