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

package apmmysql // import "go.elastic.co/apm/module/apmsql/v2/mysql"

import (
	"net"
	"strconv"

	"github.com/go-sql-driver/mysql"

	"go.elastic.co/apm/module/apmsql/v2"
)

// ParseDSN parses the given go-sql-driver/mysql datasource name.
func ParseDSN(name string) apmsql.DSNInfo {
	cfg, err := mysql.ParseDSN(name)
	if err != nil {
		// mysql.Open will fail with the same error,
		// so just return a zero value.
		return apmsql.DSNInfo{}
	}
	var addr string
	var port int
	if cfg.Net == "tcp" {
		host, portstr, _ := net.SplitHostPort(cfg.Addr)
		port, _ = strconv.Atoi(portstr)
		addr = host
	}
	return apmsql.DSNInfo{
		Address:  addr,
		Port:     port,
		Database: cfg.DBName,
		User:     cfg.User,
	}
}
