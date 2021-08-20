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

//go:build go1.12
// +build go1.12

package apmfasthttp // import "go.elastic.co/apm/module/apmfasthttp"

import (
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm"
)

type apmHandler struct {
	requestHandler   fasthttp.RequestHandler
	tracer           *apm.Tracer
	requestName      RequestNameFunc
	requestIgnorer   RequestIgnorerFunc
	recovery         RecoveryFunc
	panicPropagation bool
}

// txCloser wraps the APM transaction to implement
// the `io.Closer` interface to end the transaction automatically,
// due to it will be saved on the RequestCtx.UserValues
// which will end the transaction automatically when the request finish.
type txCloser struct {
	ctx *fasthttp.RequestCtx
	tx  *apm.Transaction
	bc  *apm.BodyCapturer
}

// ServerOption sets options for tracing server requests.
type ServerOption func(*apmHandler)

// RequestNameFunc is the type of a function for use in
// WithServerRequestName.
type RequestNameFunc func(*fasthttp.RequestCtx) string

// RequestIgnorerFunc is the type of a function for use in
// WithServerRequestIgnorer.
type RequestIgnorerFunc func(*fasthttp.RequestCtx) bool

// RecoveryFunc is the type of a function for use in WithRecovery.
type RecoveryFunc func(ctx *fasthttp.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer, recovered interface{})
