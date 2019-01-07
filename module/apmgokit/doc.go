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

// Package apmgokit provides examples and integration tests
// for tracing services implemented with Go kit.
//
// We do not provide any Go kit specific code, as the other
// generic modules (apmhttp and apmgrpc) are sufficient.
//
// Go kit-based HTTP servers can be traced by instrumenting
// the kit/transport/http.Server with apmhttp.Wrap, and HTTP
// clients can be traced by providing a net/http.Client
// instrumented with apmhttp.WrapClient.
//
// Go kit-based gRPC servers and clients can both be wrapped
// using the interceptors provided in module/apmgrpc.
package apmgokit
