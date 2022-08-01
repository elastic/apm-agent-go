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

package apmazure // import "go.elastic.co/apm/module/apmazure/v2"

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-pipeline-go/pipeline"
)

type queueRPC struct {
	accountName  string
	resourceName string
	req          pipeline.Request
	queueName    string
}

func (q *queueRPC) name() string {
	return fmt.Sprintf("AzureQueue %s %s %s", q.operation(), q.dir(), q.accountName)
}

func (q *queueRPC) _type() string {
	return "messaging"
}

func (q *queueRPC) subtype() string {
	return "azurequeue"
}

func (q *queueRPC) targetName() string {
	return q.queueName
}

func (q *queueRPC) storageAccountName() string {
	return q.accountName
}

func (q *queueRPC) resource() string {
	return q.resourceName
}

func (q *queueRPC) dir() string {
	switch q.req.Method {
	case http.MethodGet, "":
		return "from"
	default:
		return "to"
	}
}

func (q *queueRPC) operation() string {
	query := q.req.URL.Query()
	switch q.req.Method {
	// From net/http documentation:
	// For client requests, an empty string means GET.
	case http.MethodGet, "":
		return q.getOperation(query)
	case http.MethodPost:
		return q.postOperation(query)
	case http.MethodHead:
		return q.headOperation(query)
	case http.MethodPut:
		return q.putOperation(query)
	case http.MethodOptions:
		return "PREFLIGHT"
	case http.MethodDelete:
		if strings.HasSuffix(q.req.URL.Path, "/messages") {
			return "CLEAR"
		}
		return "DELETE"
	default:
		return q.req.Method
	}
}

func (q *queueRPC) getOperation(v url.Values) string {
	if peekOnly := v.Get("peekonly"); peekOnly == "true" {
		return "PEEK"
	}
	switch comp := v.Get("comp"); comp {
	case "":
		return "RECEIVE"
	case "list":
		return "LISTQUEUES"
	case "stats":
		return "STATS"
	case "properties", "metadata", "acl":
		return "GET" + strings.ToUpper(comp)
	default:
		return "unknown operation"
	}
}

func (q *queueRPC) postOperation(v url.Values) string {
	return "SEND"
}

func (q *queueRPC) headOperation(v url.Values) string {
	switch comp := v.Get("comp"); comp {
	case "metadata", "acl":
		return "GET" + strings.ToUpper(comp)
	default:
		return "unknown operation"
	}
}

func (q *queueRPC) putOperation(v url.Values) string {
	if _, ok := v["popreceipt"]; ok {
		return "UPDATE"
	}
	switch comp := v.Get("comp"); comp {
	case "":
		return "CREATE"
	case "metadata", "acl", "properties":
		return "SET" + strings.ToUpper(comp)
	default:
		return "unknown operation"
	}
}
