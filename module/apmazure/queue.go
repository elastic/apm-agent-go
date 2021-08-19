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

package apmazure // import "go.elastic.co/apm/module/apmazure"

import (
	"fmt"

	"github.com/Azure/azure-pipeline-go/pipeline"
)

type queueRPC struct {
	accountName  string
	resourceName string
	req          pipeline.Request
}

func (q *queueRPC) name() string {
	return fmt.Sprintf("AzureQueue %s %s", q.operation(), q.resourceName)
}

func (q *queueRPC) _type() string {
	return "messaging"
}

func (q *queueRPC) subtype() string {
	return "azurequeue"
}

func (q *queueRPC) storageAccountName() string {
	return q.accountName
}

func (q *queueRPC) resource() string {
	return q.resourceName
}

func (q *queueRPC) operation() string {
	return ""
}
