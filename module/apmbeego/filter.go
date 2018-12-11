package apmbeego

import (
	"context"
	"net/http"

	"github.com/astaxie/beego"
	beegocontext "github.com/astaxie/beego/context"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

type beegoFilterStateKey struct{}

type beegoFilterState struct {
	context *beegocontext.Context
}

func init() {
	AddFilters(beego.BeeApp.Handlers)
	WrapRecoverFunc(beego.BConfig)
}

// Middleware returns a beego.MiddleWare that traces requests and reports panics to Elastic APM.
func Middleware(opts ...apmhttp.ServerOption) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return apmhttp.Wrap(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			tx := apm.TransactionFromContext(req.Context())
			if tx != nil {
				state := &beegoFilterState{}
				defer func() {
					setTransactionContext(tx, state.context)
				}()
				req = apmhttp.RequestWithContext(
					context.WithValue(req.Context(), beegoFilterStateKey{}, state), req,
				)
			}
			h.ServeHTTP(w, req)
		}), opts...)
	}
}

// AddFilters adds required filters to handlers.
//
// This is called automatically for the default app (beego.BeeApp),
// so if you beego.Router, beego.RunWithMiddleware, etc., then you
// do not need to call AddFilters.
func AddFilters(handlers *beego.ControllerRegister) {
	handlers.InsertFilter("*", beego.BeforeStatic, beforeStatic, false)
}

// WrapRecoverFunc updates config's RecoverFunc so that panics will be reported to Elastic APM
// for traced requests. For non-traced requests, the original RecoverFunc will be called.
//
// WrapRecoverFunc is called automatically for the global config, beego.BConfig.
func WrapRecoverFunc(config *beego.Config) {
	orig := config.RecoverFunc
	config.RecoverFunc = func(context *beegocontext.Context) {
		if tx := apm.TransactionFromContext(context.Request.Context()); tx == nil {
			orig(context)
		}
	}
}

func beforeStatic(context *beegocontext.Context) {
	state, ok := context.Request.Context().Value(beegoFilterStateKey{}).(*beegoFilterState)
	if ok {
		state.context = context
	}
}

func setTransactionContext(tx *apm.Transaction, context *beegocontext.Context) {
	tx.Context.SetFramework("beego", beego.VERSION)
	if context != nil {
		if route, ok := context.Input.GetData("RouterPattern").(string); ok {
			tx.Name = context.Request.Method + " " + route
		}
	}
}
