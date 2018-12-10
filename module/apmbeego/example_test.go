package apmbeego_test

import (
	"github.com/astaxie/beego"

	"go.elastic.co/apm/module/apmbeego"
)

func ExampleMiddleware() {
	beego.Router("/", &testController{})
	beego.Router("/thing/:id:int", &testController{}, "get:Get")
	beego.RunWithMiddleWares("localhost:8080", apmbeego.Middleware())
}
