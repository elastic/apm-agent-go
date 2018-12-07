package apmecho_test

import (
	"github.com/labstack/echo"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmecho"
)

func ExampleMiddleware() {
	e := echo.New()
	e.Use(apmecho.Middleware())

	e.GET("/hello/:name", func(c echo.Context) error {
		// The request context contains an apm.Transaction,
		// so spans can be reported by passing the context
		// to apm.StartSpan.
		span, _ := apm.StartSpan(c.Request().Context(), "work", "custom")
		defer span.End()
		return nil
	})
}
