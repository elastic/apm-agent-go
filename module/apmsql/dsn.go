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

// DSNInfo contains information from a database-specific data source name.
type DSNInfo struct {
	// Address is the database server address specified by the DSN.
	Address string

	// Port is the database server port specified by the DSN.
	Port int

	// Database is the name of the specific database identified by the DSN.
	Database string

	// User is the username that the DSN specifies for authenticating the
	// database connection.
	User string
}

// DSNParserFunc is the type of a function that can be used for parsing a
// data source name, and returning the corresponding Info.
type DSNParserFunc func(dsn string) DSNInfo

func genericDSNParser(string) DSNInfo {
	return DSNInfo{}
}
