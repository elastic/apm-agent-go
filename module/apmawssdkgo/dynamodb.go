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

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo"

import (
	"errors"
	"reflect"

	"go.elastic.co/apm"

	"github.com/aws/aws-sdk-go/aws/request"
)

type dynamoDB struct {
	TableName string
	// KeyConditionExpression is only available on Query operations.
	KeyConditionExpression string

	name, region string
}

func (d *dynamoDB) spanName() string {
	return d.name
}

func (d *dynamoDB) resource() string {
	return d.TableName
}

func (d *dynamoDB) setAdditional(span *apm.Span) {
	dbSpanCtx := apm.DatabaseSpanContext{
		Instance: d.region,
		Type:     serviceDynamoDB,
	}
	if span.Action == "Query" {
		dbSpanCtx.Statement = d.KeyConditionExpression
	}
	span.Context.SetDatabase(dbSpanCtx)
}

func newDynamoDB(req *request.Request) (*dynamoDB, error) {
	values, err := parseDynamoDBParams(req)
	if err != nil {
		return nil, err
	}
	if r := req.Config.Region; r != nil {
		values.region = *r
	}

	values.name = req.ClientInfo.ServiceID + " " + req.Operation.Name + " " + values.TableName
	return values, nil
}

// parseDynamoDBParams reads the request Params to parse out the TableName and
// KeyConditionExpression, if present.
func parseDynamoDBParams(req *request.Request) (*dynamoDB, error) {
	b := new(dynamoDB)

	params := reflect.ValueOf(req.Params).Elem()
	if v := params.FieldByName("TableName"); v.IsValid() {
		if n, ok := v.Interface().(*string); ok {
			b.TableName = *n
		} else {
			return nil, errors.New("could not parse TableName")
		}
	} else {
		return nil, errors.New("required field TableName not present")
	}

	if v := params.FieldByName("KeyConditionExpression"); v.IsValid() {
		if n, ok := v.Interface().(*string); ok {
			b.KeyConditionExpression = *n
		}
	}

	return b, nil
}
