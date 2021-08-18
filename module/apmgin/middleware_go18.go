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

//go:build !go1.10
// +build !go1.10

package apmgin // import "go.elastic.co/apm/module/apmgin"

import (
	"sync"

	"github.com/gin-gonic/gin"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

type middleware struct {
	engine         *gin.Engine
	tracer         *apm.Tracer
	requestIgnorer apmhttp.RequestIgnorerFunc

	setRouteMapOnce sync.Once
	routeMap        map[string]map[string]routeInfo
}

type routeInfo struct {
	transactionName string // e.g. "GET /foo"
}

func (m *middleware) getRequestName(c *gin.Context) string {
	// NOTE(axw) this implementation exists for older versions of Go (<1.10)
	// which cannot use the version of Gin with `c.FullPath`. This is broken
	// when the same handler is used for multiple routes.
	m.setRouteMapOnce.Do(func() {
		routes := m.engine.Routes()
		rm := make(map[string]map[string]routeInfo)
		for _, r := range routes {
			mm := rm[r.Method]
			if mm == nil {
				mm = make(map[string]routeInfo)
				rm[r.Method] = mm
			}
			mm[r.Handler] = routeInfo{
				transactionName: r.Method + " " + r.Path,
			}
		}
		m.routeMap = rm
	})
	if routeInfo, ok := m.routeMap[c.Request.Method][c.HandlerName()]; ok {
		return routeInfo.transactionName
	}
	return apmhttp.UnknownRouteRequestName(c.Request)
}
