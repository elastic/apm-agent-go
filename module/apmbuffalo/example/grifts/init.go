package grifts

import (
	"github.com/gobuffalo/buffalo"

	"go.elastic.co/apm/module/apmbuffalo/example/actions"
)

func init() {
	buffalo.Grifts(actions.App())
}
