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

package apmgorm_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmgorm"
	_ "go.elastic.co/apm/module/apmgorm/dialects/mysql"
	_ "go.elastic.co/apm/module/apmgorm/dialects/postgres"
	_ "go.elastic.co/apm/module/apmgorm/dialects/sqlite"
	"go.elastic.co/apm/module/apmsql"
)

type Product struct {
	gorm.Model
	Code  string
	Price uint
}

func TestWithContext(t *testing.T) {
	t.Run("sqlite3", func(t *testing.T) {
		testWithContext(t,
			apmsql.DSNInfo{Database: ":memory:"},
			"sqlite3", ":memory:",
		)
	})

	if os.Getenv("PGHOST") == "" {
		t.Logf("PGHOST not specified, skipping")
	} else {
		t.Run("postgres", func(t *testing.T) {
			testWithContext(t,
				apmsql.DSNInfo{Database: "test_db", User: "postgres"},
				"postgres", "user=postgres password=hunter2 dbname=test_db sslmode=disable",
			)
		})
	}

	if mysqlHost := os.Getenv("MYSQL_HOST"); mysqlHost == "" {
		t.Logf("MYSQL_HOST not specified, skipping")
	} else {
		t.Run("mysql", func(t *testing.T) {
			testWithContext(t,
				apmsql.DSNInfo{Database: "test_db", User: "root"},
				"mysql", "root:hunter2@tcp("+mysqlHost+")/test_db?parseTime=true",
			)
		})
	}
}

func testWithContext(t *testing.T, dsnInfo apmsql.DSNInfo, dialect string, args ...interface{}) {
	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		db, err := apmgorm.Open(dialect, args...)
		require.NoError(t, err)
		defer db.Close()
		db = apmgorm.WithContext(ctx, db)

		db.AutoMigrate(&Product{})
		db.Create(&Product{Code: "L1212", Price: 1000})

		var product Product
		assert.NoError(t, db.First(&product, "code = ?", "L1212").Error)
		assert.NoError(t, db.Model(&product).Update("Price", 2000).Error)
		assert.NoError(t, db.Delete(&product).Error)            // soft
		assert.NoError(t, db.Unscoped().Delete(&product).Error) // hard
	})
	require.NotEmpty(t, spans)
	assert.Empty(t, errors)

	spanNames := make([]string, len(spans))
	for i, span := range spans {
		spanNames[i] = span.Name
		require.NotNil(t, span.Context)
		require.NotNil(t, span.Context.Database)
		assert.Equal(t, dsnInfo.Database, span.Context.Database.Instance)
		assert.NotEmpty(t, span.Context.Database.Statement)
		assert.Equal(t, "sql", span.Context.Database.Type)
		assert.Equal(t, dsnInfo.User, span.Context.Database.User)
	}
	assert.Equal(t, []string{
		"INSERT INTO products",
		"SELECT FROM products",
		"UPDATE products",
		"UPDATE products", // soft delete
		"DELETE FROM products",
	}, spanNames)
}

// TestWithContextNoTransaction checks that using WithContext without
// a transaction won't cause any issues.
func TestWithContextNoTransaction(t *testing.T) {
	db, err := apmgorm.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	db = apmgorm.WithContext(context.Background(), db)

	db.AutoMigrate(&Product{})
	db.Create(&Product{Code: "L1212", Price: 1000})

	var product Product
	assert.NoError(t, db.Where("code=?", "L1212").First(&product).Error)
}

func TestWithContextNonSampled(t *testing.T) {
	os.Setenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE", "0")
	defer os.Unsetenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE")

	db, err := apmgorm.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	db.AutoMigrate(&Product{})

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		db = apmgorm.WithContext(ctx, db)
		db.Create(&Product{Code: "L1212", Price: 1000})
	})
	require.Empty(t, spans)
}

func TestCaptureErrors(t *testing.T) {
	db, err := apmgorm.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	db.SetLogger(nopLogger{})
	db.AutoMigrate(&Product{})

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		db = apmgorm.WithContext(ctx, db)

		// record not found should not cause an error
		db.Where("code=?", "L1212").First(&Product{})

		// invalid SQL should
		db.Where("bananas").First(&Product{})
	})
	assert.Len(t, spans, 2)
	require.Len(t, errors, 1)
	assert.Regexp(t, "no such column: bananas", errors[0].Exception.Message)
}

func TestOpenWithDriver(t *testing.T) {
	db, err := apmgorm.Open("sqlite3", "sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	db.AutoMigrate(&Product{})

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		db = apmgorm.WithContext(ctx, db)
		db.Create(&Product{Code: "L1212", Price: 1000})
	})
	require.Len(t, spans, 1)
	assert.Equal(t, ":memory:", spans[0].Context.Database.Instance)
}

func TestOpenWithDB(t *testing.T) {
	sqldb, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sqldb.Close()

	db, err := apmgorm.Open("sqlite3", sqldb)
	require.NoError(t, err)
	defer db.Close()
	db.AutoMigrate(&Product{})

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		db = apmgorm.WithContext(ctx, db)
		db.Create(&Product{Code: "L1212", Price: 1000})
	})
	require.Len(t, spans, 1)
	assert.Empty(t, spans[0].Context.Database.Instance) // no DSN info
}

type nopLogger struct{}

func (nopLogger) Print(v ...interface{}) {}
