module go.elastic.co/apm/module/apmprometheus

require (
	github.com/pkg/errors v0.8.0
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.2.0
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4 // indirect
)

replace go.elastic.co/apm => ../..
