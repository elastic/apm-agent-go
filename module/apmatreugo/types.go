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

// +build go1.12

package apmatreugo // import "go.elastic.co/apm/module/apmatreugo"

import (
	"github.com/savsgio/atreugo/v11"

	"go.elastic.co/apm"
)

// Factory is a factory to create the tracing middleware and panic view.
type Factory struct {
	tracer           *apm.Tracer
	requestName      RequestNameFunc
	requestIgnorer   RequestIgnorerFunc
	recovery         RecoveryFunc
	panicPropagation bool
}

// Option sets options for tracing requests.
type Option func(*Factory)

// RequestNameFunc is the type of a function for use in
// WithServerRequestName.
type RequestNameFunc func(*atreugo.RequestCtx) string

// RequestIgnorerFunc is the type of a function for use in
// WithServerRequestIgnorer.
type RequestIgnorerFunc func(*atreugo.RequestCtx) bool

// RecoveryFunc is the type of a function for use in WithRecovery.
type RecoveryFunc func(ctx *atreugo.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer, recovered interface{})
