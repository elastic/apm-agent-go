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
package apmbuffalo_test

import (
	"errors"
	"github.com/gobuffalo/mw-contenttype"
	"log"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/x/sessions"
	"github.com/rs/cors"

	"go.elastic.co/apm/module/apmbuffalo"
)

var r *render.Engine
func init() {
	r = render.New(render.Options{
		DefaultContentType: "application/json",
	})
}

// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c buffalo.Context) error {
	return c.Render(200, r.JSON(map[string]string{"message": "Welcome to Buffalo!"}))
}

func PanicHandler(c buffalo.Context) error {
	panic("Aaaaouch")
}

func ErrorHandler(c buffalo.Context) error {
	return c.Error(499, errors.New("error 499"))
}

var ENV = envy.Get("GO_ENV", "development")
var app *buffalo.App

func App() *buffalo.App {
	if app == nil {
		app = buffalo.New(buffalo.Options{
			Env:          ENV,
			SessionStore: sessions.Null{},
			PreWares: []buffalo.PreWare{
				cors.Default().Handler,
			},
			SessionName: "_example_session",
		})

		// Set the request content type to JSON
		app.Use(contenttype.Set("application/json"))

		app.GET("/", HomeHandler)
		app.GET("/panic", PanicHandler)
		app.GET("/error", ErrorHandler)
	}

	return app
}

// main is the starting point for your Buffalo application.
// You can feel free and add to this `main` method, change
// what it does, etc...
// All we ask is that, at some point, you make sure to
// call `app.Serve()`, unless you don't want to start your
// application that is. :)
func main() {
	app := App()
	apmbuffalo.Instrument(app)
	if err := app.Serve(); err != nil {
		log.Fatal(err)
	}
}