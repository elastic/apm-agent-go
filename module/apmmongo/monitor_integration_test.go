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

package apmmongo_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmmongo"
)

var (
	mongoURL = os.Getenv("MONGO_URL")
)

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

type IntegrationSuite struct {
	suite.Suite
	client *mongo.Client
}

func (suite *IntegrationSuite) SetupSuite() {
	if mongoURL == "" {
		suite.T().Skipf("MONGO_URL not specified")
	}
	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(mongoURL).SetMonitor(apmmongo.CommandMonitor()).SetAuth(
			options.Credential{
				Username: "admin",
				Password: "hunter2",
			},
		),
	)
	suite.NoError(err)
	suite.client = client
}

func (suite *IntegrationSuite) TearDownSuite() {
	err := suite.client.Disconnect(context.Background())
	suite.NoError(err)
}

func (suite *IntegrationSuite) TestCommandMonitor() {
	tx, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		db := suite.client.Database("test_db")
		coll := db.Collection("test_coll")

		err := coll.Drop(ctx)
		suite.Require().NoError(err)

		err = db.RunCommand(ctx, bson.D{{Key: "dropUser", Value: "bob"}}).Err()
		apm.CaptureError(ctx, err).Send() // User not found

		_, err = coll.InsertMany(ctx, []interface{}{
			bson.D{{Key: "foo", Value: "bar"}},
			bson.D{{Key: "baz", Value: "qux"}},
		})
		suite.Require().NoError(err)

		cur, err := coll.Find(ctx, &bson.D{{Key: "foo", Value: "bar"}})
		suite.Require().NoError(err)
		defer cur.Close(ctx)

		var n int
		for cur.Next(ctx) {
			n++
		}
		suite.Equal(1, n)
		suite.NoError(cur.Err())
	})

	suite.Require().Len(spans, 5)
	suite.Equal("test_coll.drop", spans[0].Name)
	suite.Equal("dropUser", spans[1].Name)
	suite.Equal("test_coll.insert", spans[2].Name)
	suite.Equal("test_coll.find", spans[3].Name)
	suite.Equal("test_coll.killCursors", spans[4].Name)

	// We capture the command body as Extended JSON.
	suite.Equal(`{"drop":"test_coll","$db":"test_db"}`, spans[0].Context.Database.Statement)

	suite.Require().Len(errs, 1)
	suite.Equal(tx.ID, errs[0].ParentID)
	suite.Equal("(UserNotFound) User 'bob@test_db' not found", errs[0].Exception.Message)
	suite.Equal(model.ExceptionCode{String: "UserNotFound"}, errs[0].Exception.Code)
}
