module go.elastic.co/apm/module/apmchi

require (
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/prometheus/procfs v0.0.0-20190328153300-af7bedc223fb // indirect
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmhttp v1.4.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
