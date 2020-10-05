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

// +build go1.9

package apmgormv2

import (
	"go.elastic.co/apm/module/apmsql"
	"gorm.io/gorm"
)

// Plugin struct
// - It can be used with existing *gorm.DB using db.Use(NewPlugin())
type Plugin struct {
}

// NewPlugin plugin constructor
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name name of plugin
func (p Plugin) Name() string {
	return "elasticapm"
}

// Initialize to register callbacks
func (p Plugin) Initialize(db *gorm.DB) error {
	dialect := db.Dialector
	dsn, err := extractDsn(dialect)
	if err != nil {
		return err
	}
	registerCallbacks(db, apmsql.DriverDSNParser(dialect.Name())(dsn))
	return nil
}
