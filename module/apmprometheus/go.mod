module go.elastic.co/apm/module/apmprometheus/v2

require (
	github.com/elastic/go-licenser v0.4.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/tools v0.1.8 // indirect
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
