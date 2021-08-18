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

package apmgopg_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmgopg"
)

type User struct {
	TableName struct{} `sql:"users"`

	Id   int    `json:"id"`
	Name string `json:"name"`
}

func TestWithContext(t *testing.T) {
	if os.Getenv("PGHOST") == "" {
		t.Skip("PGHOST env not defined, skipping")
		return
	}

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		db := pg.Connect(&pg.Options{
			Addr:     fmt.Sprintf("%s:5432", os.Getenv("PGHOST")),
			User:     "postgres",
			Password: "hunter2",
			Database: "test_db",
		})
		err := apmgopg.Instrument(db)
		require.NoError(t, err)

		_, err = db.Exec("SELECT 1")
		require.NoError(t, err)

		db.DropTable(&User{}, &orm.DropTableOptions{})
		db.CreateTable(&User{}, &orm.CreateTableOptions{})

		defer db.Close()
		db = db.WithContext(ctx)

		_, err = db.Model(&User{Id: 1337, Name: "Antoine Hedgecock"}).Insert()
		assert.NoError(t, err)

		assert.NoError(t, db.Model(&User{}).Where("id = ?", 1337).Select())

		_, err = db.Model(&User{Id: 1337, Name: "new name"}).Column("name").WherePK().Update()
		assert.NoError(t, err)
	})

	require.NotEmpty(t, spans)
	assert.Empty(t, errors)

	spanNames := make([]string, len(spans))
	for i, span := range spans {
		spanNames[i] = span.Name
		require.NotNil(t, span.Context)
		require.NotNil(t, span.Context.Database)
		assert.Equal(t, "test_db", span.Context.Database.Instance)
		assert.NotEmpty(t, span.Context.Database.Statement)
		assert.Equal(t, "sql", span.Context.Database.Type)
		assert.Equal(t, "postgres", span.Context.Database.User)
	}

	assert.Equal(t, []string{
		"INSERT INTO users",
		"SELECT FROM users",
		"UPDATE users",
	}, spanNames)
}
