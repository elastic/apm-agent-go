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

package apmazure

import (
	"bytes"
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

type tokenCredential struct{}

func (t *tokenCredential) GetToken(_ context.Context, _ azcore.TokenRequestOptions) (*azcore.AccessToken, error) {
	return nil, nil
}

func (t *tokenCredential) AuthenticationPolicy(options azcore.AuthenticationPolicyOptions) azcore.Policy {
	return new(fakePolicy)
}

type fakePolicy struct{}

func (p *fakePolicy) Do(req *azcore.Request) (*azcore.Response, error) {
	resp, err := req.Next()
	return resp, err
}

type fakeTransport struct{}

func (t *fakeTransport) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       &fakeBuffer{new(bytes.Buffer)},
	}, nil
}

type fakeBuffer struct{ *bytes.Buffer }

func (b *fakeBuffer) Close() error { return nil }
