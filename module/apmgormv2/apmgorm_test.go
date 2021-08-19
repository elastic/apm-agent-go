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

package apmgormv2_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"gorm.io/gorm/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"go.elastic.co/apm/apmtest"
	mysql "go.elastic.co/apm/module/apmgormv2/driver/mysql"
	postgres "go.elastic.co/apm/module/apmgormv2/driver/postgres"
	sqlite "go.elastic.co/apm/module/apmgormv2/driver/sqlite"
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
			sqlite.Open(":memory:"), &gorm.Config{},
		)
	})

	if pgHost := os.Getenv("PGHOST"); pgHost == "" {
		t.Logf("PGHOST not specified, skipping")
	} else {
		t.Run("postgres", func(t *testing.T) {
			testWithContext(t,
				apmsql.DSNInfo{
					Address:  pgHost,
					Port:     5432,
					Database: "test_db",
					User:     "postgres",
				},
				postgres.Open("user=postgres password=hunter2 dbname=test_db sslmode=disable"), &gorm.Config{},
			)
		})
	}

	if mysqlHost := os.Getenv("MYSQL_HOST"); mysqlHost == "" {
		t.Logf("MYSQL_HOST not specified, skipping")
	} else {
		t.Run("mysql", func(t *testing.T) {
			testWithContext(t,
				apmsql.DSNInfo{
					Address:  mysqlHost,
					Port:     3306,
					Database: "test_db",
					User:     "root",
				},
				mysql.Open("root:hunter2@tcp("+mysqlHost+")/test_db?parseTime=true"), &gorm.Config{},
			)
		})
	}
}

func testWithContext(t *testing.T, dsnInfo apmsql.DSNInfo, dialect gorm.Dialector, config *gorm.Config) {
	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		db, err := gorm.Open(dialect, config)
		require.NoError(t, err)
		ddb, _ := db.DB()
		defer ddb.Close()

		db = db.WithContext(ctx)

		if db.Migrator().HasTable(&Product{}) {
			db.Migrator().DropTable(&Product{})
		}
		db.AutoMigrate(&Product{})
		db.Create(&Product{Code: "L1212", Price: 1000})

		var product Product
		var count int64
		assert.NoError(t, db.Model(&product).Count(&count).Error)
		assert.Equal(t, int64(1), count)
		assert.NoError(t, db.First(&product, "code = ?", "L1212").Error)
		assert.NoError(t, db.Model(&product).Update("Price", 2000).Error)
		assert.NoError(t, db.Delete(&product).Error)            // soft
		assert.NoError(t, db.Unscoped().Delete(&product).Error) // hard
	})
	require.NotEmpty(t, spans)
	assert.Empty(t, errors)

	var spanNames []string
	for _, span := range spans {
		if strings.Contains(span.Name, "products") && span.Action != "prepare" {
			spanNames = append(spanNames, span.Name)
		}
		require.NotNil(t, span.Context)
		require.NotNil(t, span.Context.Database)
		assert.Equal(t, dsnInfo.Database, span.Context.Database.Instance)
		assert.NotEmpty(t, span.Context.Database.Statement)
		assert.Equal(t, "sql", span.Context.Database.Type)
		assert.Equal(t, dsnInfo.User, span.Context.Database.User)
		if dsnInfo.Address == "" {
			assert.Nil(t, span.Context.Destination)
		} else {
			assert.Equal(t, dsnInfo.Address, span.Context.Destination.Address)
			assert.Equal(t, dsnInfo.Port, span.Context.Destination.Port)
		}
	}
	assert.Equal(t, []string{
		"INSERT INTO products",
		"SELECT FROM products", // count
		"SELECT FROM products",
		"UPDATE products",
		"UPDATE products", // soft delete
		"DELETE FROM products",
	}, spanNames)
}

// TestWithContextNoTransaction checks that using WithContext without
// a transaction won't cause any issues.
func TestWithContextNoTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	ddb, _ := db.DB()
	defer ddb.Close()

	db = db.WithContext(context.Background())
	if db.Migrator().HasTable(&Product{}) {
		db.Migrator().DropTable(&Product{})
	}
	db.AutoMigrate(&Product{})
	db.Create(&Product{Code: "L1212", Price: 1000})

	var product Product
	assert.NoError(t, db.Where("code=?", "L1212").First(&product).Error)
}

func TestWithContextNonSampled(t *testing.T) {
	os.Setenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE", "0")
	defer os.Unsetenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	require.NoError(t, err)

	ddb, _ := db.DB()
	defer ddb.Close()

	if db.Migrator().HasTable(&Product{}) {
		db.Migrator().DropTable(&Product{})
	}
	db.AutoMigrate(&Product{})

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		db = db.WithContext(ctx)
		db.Create(&Product{Code: "L1212", Price: 1000})
	})
	require.Empty(t, spans)
}

func TestCaptureErrors(t *testing.T) {
	t.Run("sqlite3", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		ddb, _ := db.DB()
		defer ddb.Close()

		testCaptureErrors(t, db)
	})
}

func testCaptureErrors(t *testing.T, db *gorm.DB) {
	db.Config.Logger = logger.Default.LogMode(logger.Silent)
	if db.Migrator().HasTable(&Product{}) {
		db.Migrator().DropTable(&Product{})
	}
	db.AutoMigrate(&Product{})

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		db = db.WithContext(ctx)

		// gorm.ErrRecordNotFound should not cause an error
		db.Where("code=?", "L1212").First(&Product{})

		product := Product{
			Model: gorm.Model{
				ID: 1001,
			},
			Code:  "1001",
			Price: 1001,
		}
		require.NoError(t, db.Create(&product).Error)

		// invalid SQL should
		db.Where("bananas").First(&Product{})
	})
	assert.Len(t, spans, 3)
	require.Len(t, errors, 1)
	assert.Regexp(t, `.*bananas.*`, errors[0].Exception.Message)
}

func TestOpenWithDriver(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	require.NoError(t, err)

	ddb, _ := db.DB()
	defer ddb.Close()

	if db.Migrator().HasTable(&Product{}) {
		db.Migrator().DropTable(&Product{})
	}
	db.AutoMigrate(&Product{})

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		db = db.WithContext(ctx)
		db.Create(&Product{Code: "L1212", Price: 1000})
	})
	require.Len(t, spans, 1)
	assert.Equal(t, ":memory:", spans[0].Context.Database.Instance)
}
