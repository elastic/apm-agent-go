module go.elastic.co/apm/module/apmbuffalo/example

require (
	github.com/codegangsta/negroni v1.0.0 // indirect
	github.com/gobuffalo/buffalo v0.14.3
	github.com/gobuffalo/buffalo-plugins v1.14.1 // indirect
	github.com/gobuffalo/envy v1.6.15
	github.com/gobuffalo/mw-contenttype v0.0.0-20190224202710-36c73cc938f3 // indirect
	github.com/gobuffalo/mw-csrf v0.0.0-20190129204204-25460a055517 // indirect
	github.com/gobuffalo/mw-forcessl v0.0.0-20190224202501-6d1ef7ffb276
	github.com/gobuffalo/mw-paramlogger v0.0.0-20190224201358-0d45762ab655
	github.com/gobuffalo/packr v1.24.1
	github.com/gobuffalo/packr/v2 v2.1.0
	github.com/gobuffalo/pop v4.10.0+incompatible
	github.com/gobuffalo/suite v2.6.2+incompatible
	github.com/markbates/grift v1.0.5
	github.com/rs/cors v1.6.0 // indirect
	github.com/unrolled/secure v1.0.0

	go.elastic.co/apm v1.3.0
	go.elastic.co/apm/module/apmbuffalo v1.3.0
	go.elastic.co/apm/module/apmhttp v1.3.0

)

replace go.elastic.co/apm => ../../..

replace go.elastic.co/apm/module/apmbuffalo => ../

replace go.elastic.co/apm/module/apmhttp => ../../apmhttp
