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

package apmpgxv5 // import "go.elastic.co/apm/module/apmsql/v2/pgxv5"

import (
	"github.com/jackc/pgx/v5/stdlib"

	"go.elastic.co/apm/module/apmsql/v2"
	"go.elastic.co/apm/module/apmsql/v2/internal/pgutil"
)

// DriverName for pgx v5
const DriverName = apmsql.DriverPrefix + "pgx/v5"

func init() {
	apmsql.Register("pgx/v5", stdlib.GetDefaultDriver(), apmsql.WithDSNParser(pgutil.ParseDSN))
}
