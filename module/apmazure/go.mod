module go.elastic.co/apm/module/apmazure

go 1.14

require (
	github.com/Azure/azure-pipeline-go v0.2.3
	github.com/Azure/azure-storage-blob-go v0.14.0
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm v1.13.1
	go.elastic.co/apm/module/apmhttp v1.13.1
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
