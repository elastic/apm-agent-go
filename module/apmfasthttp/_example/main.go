package main

import (
	"bufio"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
	"go.elastic.co/apm/module/apmfasthttp"
)

func main() {
	h := func(ctx *fasthttp.RequestCtx) {
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			w.WriteString("Hello")
			w.WriteByte('\n')

			time.Sleep(5 * time.Second)

			w.WriteString("World")
			w.WriteByte('\n')
		})
	}

	s := fasthttp.Server{
		Handler: apmfasthttp.Wrap(h),
	}

	fmt.Println("Ready...")
	panic(s.ListenAndServe(":8000"))
}

// export ELASTIC_APM_SERVICE_NAME=r2d2_local
// export ELASTIC_APM_SERVER_URL=http://apm-dev.atani.com
