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

package apmsql // import "go.elastic.co/apm/module/apmsql/v2"

import (
	"database/sql/driver"
	"errors"
)

// namedValueToValue copied from database/sql (see NOTICE).
func namedValueToValue(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))
	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}
		dargs[n] = param.Value
	}
	return dargs, nil
}

// namedValueChecker is identical to driver.NamedValueChecker, existing
// for compatibility with Go 1.8.
type namedValueChecker interface {
	CheckNamedValue(*driver.NamedValue) error
}

func checkNamedValue(nv *driver.NamedValue, next namedValueChecker) error {
	if next != nil {
		return next.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}
