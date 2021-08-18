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

//go:build go1.14
// +build go1.14

package apmpgxv4 // import "go.elastic.co/apm/module/apmsql/pgxv4"

import (
	"github.com/jackc/pgx/v4/stdlib"

	"go.elastic.co/apm/module/apmsql/internal/pgutil"

	"go.elastic.co/apm/module/apmsql"
)

// DriverName for pgx v4
const DriverName = apmsql.DriverPrefix + "pgx"

func init() {
	apmsql.Register("pgx", &stdlib.Driver{}, apmsql.WithDSNParser(pgutil.ParseDSN))
}
