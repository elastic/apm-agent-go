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

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo/v2"

import (
	"strings"

	"go.elastic.co/apm/v2"

	"github.com/aws/aws-sdk-go/aws/request"
)

type s3 struct {
	name, bucketName string
}

func newS3(req *request.Request) *s3 {
	bucketName := getBucketName(req)
	name := req.ClientInfo.ServiceID + " " + req.Operation.Name + " " + bucketName
	return &s3{name: name, bucketName: bucketName}
}

func (s *s3) spanName() string {
	return s.name
}

func (s *s3) resource() string {
	return s.bucketName
}

func (s *s3) targetName() string {
	return s.bucketName
}

func (s *s3) setAdditional(*apm.Span) {}

// getBucketName extracts the bucket name from the URL generated by the AWS SDK
// for communicating with S3. By default, the SDK will use the virtual bucket
// naming scheme, http://BUCKET.s3.amazonaws.com/KEY, to communicate with S3.
// Regardless of configuration, some routes will rely on the path style,
// http://s3.amazonaws.com/BUCKET/KEY.
// https://github.com/aws/aws-sdk-go/blob/d9428afe4490b19/aws/config.go#L118-L127
func getBucketName(req *request.Request) string {
	host := req.HTTPRequest.URL.Host
	if strings.HasPrefix(host, req.ClientInfo.ServiceName) {
		return strings.Split(req.HTTPRequest.URL.Path[1:], "/")[0]
	}
	return strings.Split(req.HTTPRequest.URL.Host, ".")[0]
}
