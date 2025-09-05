---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/supported-tech.html
applies_to:
  stack:
  serverless:
    observability:
  product:
    apm_agent_go: ga
---

# Supported technologies [supported-tech]

This page describes the technologies supported by the Elastic APM Go agent.

If your favorite technology is not supported yet, you start a conversation in the [Discuss forum](https://discuss.elastic.co/c/apm).

If you would like to get more involved, take a look at the [contributing guide](/reference/contributing.md).


## Go [supported-tech-go]

The Elastic APM Go agent naturally requires Go. We support the last two major Go releases as described by [Go’s Release Policy](https://golang.org/doc/devel/release.md#policy):

Each major Go release is supported until there are two newer major releases. For example, Go 1.5 was supported until the Go 1.7 release, and Go 1.6 was supported until the Go 1.8 release.

## Web Frameworks [supported-tech-web-frameworks]

We support several third-party web frameworks, as well as Go’s standard `net/http` package. Regardless of the framework, we create a transaction for each incoming request, and name the transaction after the registered route.


### fasthttp [_fasthttp]

We support [valyala/fasthttp](https://github.com/valyala/fasthttp), [v1.26.0](https://github.com/valyala/fasthttp/releases/tag/v1.26.0) and greater.

See [module/apmfasthttp](/reference/builtin-modules.md#builtin-modules-apmfasthttp) for more information about fasthttp instrumentation.


### httprouter [_httprouter]

[julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) does not use semantic versioning, but its API is relatively stable. Any recent version should be compatible with the Elastic APM Go agent.

See [module/apmhttprouter](/reference/builtin-modules.md#builtin-modules-apmhttprouter) for more information about httprouter instrumentation.


### Echo [_echo]

We support the [Echo](https://echo.labstack.com/) web framework, [v3.3.5](https://github.com/labstack/echo/releases/tag/3.3.5) and greater.

We provide different packages for the Echo v3 and v4 versions: `module/apmecho` for Echo v3.x, and `module/apmechov4` for Echo v4.x.

See [module/apmecho](/reference/builtin-modules.md#builtin-modules-apmecho) for more information about Echo instrumentation.


### Gin [_gin]

We support the [Gin](https://gin-gonic.com/) web framework, [v1.2](https://github.com/gin-gonic/gin/releases/tag/v1.2) and greater.

See [module/apmgin](/reference/builtin-modules.md#builtin-modules-apmgin) for more information about Gin instrumentation.


### Fiber [_fiber]

We support the [Fiber](https://gofiber.io/) web framework, [v2.18.0](https://github.com/gofiber/fiber/releases/tag/v2.18.0) and greater.

We provide package only for the Fiber v2. See [module/apmfiber](/reference/builtin-modules.md#builtin-modules-apmfiber) for more information about Fiber instrumentation.


### Beego [_beego]

We support the [Beego](https://beego.me/) web framework, [v1.10.0](https://github.com/astaxie/beego/releases/tag/v1.10.0) and greater.

See [module/apmbeego](/reference/builtin-modules.md#builtin-modules-apmbeego) for more information about Beego instrumentation.


### gorilla/mux [_gorillamux]

We support [gorilla/mux](http://www.gorillatoolkit.org/pkg/mux) [v1.6.1](https://github.com/gorilla/mux/releases/tag/v1.6.1) and greater. Older versions are not supported due to the use of gorilla.Middleware.

See [module/apmgorilla](/reference/builtin-modules.md#builtin-modules-apmgorilla) for more information about gorilla/mux instrumentation.


### go-restful [_go_restful]

We support [go-restful](https://github.com/emicklei/go-restful), [2.0.0](https://github.com/emicklei/go-restful/releases/tag/2.0.0) and greater.

See [module/apmrestful](/reference/builtin-modules.md#builtin-modules-apmrestful) for more information about go-restful instrumentation.


### chi [_chi]

We support [chi](https://github.com/go-chi/chi), [v4.0.0](https://github.com/go-chi/chi/releases/tag/v4.0.0) and greater.

See [module/apmchi](/reference/builtin-modules.md#builtin-modules-apmchi) for more information about chi instrumentation.


### negroni [_negroni]

We support [negroni](https://github.com/urfave/negroni), [v1.0.0](https://github.com/urfave/negroni/releases/tag/v1.0.0) and greater.

See [module/apmnegroni](/reference/builtin-modules.md#builtin-modules-apmnegroni) for more information about negroni instrumentation.


## Databases [supported-tech-databases]


### database/sql [_databasesql]

We support tracing requests with any `database/sql` driver, provided the driver is registered with the Elastic APM Go agent. Spans will be created for each statemented executed.

When using one of the following drivers, the Elastic APM Go agent will be able to parse the datasource name, and provide more context in the spans it emits:

* [lib/pq](https://github.com/lib/pq) (PostgreSQL)
* [jackc/pgx](https://github.com/jackc/pgx) (PostgreSQL)
* [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
* [mattn/go-sqlite3](https://github.com/go-sqlite3)

See [module/apmsql](/reference/builtin-modules.md#builtin-modules-apmsql) for more information about database/sql instrumentation.


### GORM [_gorm]

We support the [GORM](http://gorm.io/) object-relational mapping library, [v1.9](https://github.com/jinzhu/gorm/releases/tag/v1.9) and greater. Spans will be created for each create, query, update, and delete operation.

As with `database/sql` support we provide additional support for the postgres, mysql, and sqlite dialects.

We provide different packages for the Gorm v1 and v2 versions: `module/apmgorm` for Gorm v1.x, and `module/apmgormv2` for Gorm v2.x.

See [module/apmgorm](/reference/builtin-modules.md#builtin-modules-apmgorm) or [module/apmgormv2](/reference/builtin-modules.md#builtin-modules-apmgorm) for more information about GORM instrumentation.


### go-pg/pg [_go_pgpg]

We support the [go-pg/pg](https://github.com/go-pg/pg) PostgreSQL ORM, [v8.0.4](https://github.com/go-pg/pg/releases/tag/v8.0.4). Spans will be created for each database operation.

See [module/apmgopg](/reference/builtin-modules.md#builtin-modules-apmgopg) for more information about go-pg instrumentation.


### Cassandra (gocql) [_cassandra_gocql]

[GoCQL](https://gocql.github.io/) does not have a stable API, so we will provide support for the most recent API, and older versions of the API on a best-effort basis. Spans will be created for each query. When the batch API is used, a span will be created for the batch, and a sub-span is created for each query in the batch.

See [module/apmgocql](/reference/builtin-modules.md#builtin-modules-apmgocql) for more information about GoCQL instrumentation.


### Redis (gomodule/redigo) [_redis_gomoduleredigo]

We support [Redigo](https://github.com/gomodule/redigo), [v2.0.0](https://github.com/gomodule/redigo/tree/v2.0.0) and greater. We provide helper functions for reporting Redis commands as spans.

See [module/apmredigo](/reference/builtin-modules.md#builtin-modules-apmredigo) for more information about Redigo instrumentation.


### Redis (go-redis/redis) [_redis_go_redisredis]

We support [go-redis](https://github.com/go-redis/redis), [v6.15.3](https://github.com/go-redis/redis/tree/v6.15.3). We provide helper functions for reporting Redis commands as spans.

See [module/apmgoredis](/reference/builtin-modules.md#builtin-modules-apmgoredis) for more information about go-redis instrumentation.


### Elasticsearch [_elasticsearch]

We provide instrumentation for Elasticsearch clients. This is usable with the [go-elasticsearch](https://github.com/elastic/go-elasticsearch) and [olivere/elastic](https://github.com/olivere/elastic) clients, and should also be usable with any other clients that provide a means of configuring the underlying `net/http.RoundTripper`.

See [module/apmelasticsearch](/reference/builtin-modules.md#builtin-modules-apmelasticsearch) for more information about Elasticsearch client instrumentation.


### MongoDB [_mongodb]

We provide instrumentation for the official [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver), [v1.0.0](https://github.com/mongodb/mongo-go-driver/releases/tag/v1.0.0) and greater. Spans will be created for each MongoDB command executed within a context containing a transaction.

See [module/apmmongo](/reference/builtin-modules.md#builtin-modules-apmmongo) for more information about the MongoDB Go Driver instrumentation.


### DynamoDB [_dynamodb]

We provide instrumentation for AWS DynamoDB. This is usable with [AWS SDK Go](https://github.com/aws/aws-sdk-go).

See [module/apmawssdkgo](/reference/builtin-modules.md#builtin-modules-apmawssdkgo) for more information about AWS SDK Go instrumentation.


## RPC Frameworks [supported-tech-rpc]


### gRPC [_grpc]

We support [gRPC](https://grpc.io/) [v1.3.0](https://github.com/grpc/grpc-go/releases/tag/v1.3.0) and greater. We provide unary and stream interceptors for both the client and server. The server interceptor will create a transaction for each incoming request, and the client interceptor will create a span for each outgoing request.

See [module/apmgrpc](/reference/builtin-modules.md#builtin-modules-apmgrpc) for more information about gRPC instrumentation.


## Service Frameworks [supported-tech-services]


### Go kit [_go_kit]

We support tracing [Go kit](https://gokit.io/) clients and servers when using the gRPC or HTTP transport, by way of [module/apmgrpc](/reference/builtin-modules.md#builtin-modules-apmgrpc) and [module/apmhttp](/reference/builtin-modules.md#builtin-modules-apmhttp) respectively.

Code examples are available at [https://pkg.go.dev/go.elastic.co/apm/module/apmgokit/v2](https://pkg.go.dev/go.elastic.co/apm/module/apmgokit/v2) for getting started.


## Logging frameworks [supported-tech-logging]


### Logrus [_logrus]

We support log correlation and exception tracking with [Logrus](https://github.com/sirupsen/logrus/), [v1.1.0](https://github.com/sirupsen/logrus/releases/tag/v1.1.0) and greater.

See [module/apmlogrus](/reference/builtin-modules.md#builtin-modules-apmlogrus) for more information about Logrus integration.


### Zap [_zap]

We support log correlation and exception tracking with [Zap](https://github.com/uber-go/zap/), [v1.0.0](https://github.com/uber-go/zap/releases/tag/v1.0.0) and greater.

See [module/apmzap](/reference/builtin-modules.md#builtin-modules-apmzap) for more information about Zap integration.


### Zerolog [_zerolog]

We support log correlation and exception tracking with [Zerolog](https://github.com/rs/zerolog/), [v1.12.0](https://github.com/rs/zerolog/releases/tag/v1.12.0) and greater.

See [module/apmzerolog](/reference/builtin-modules.md#builtin-modules-apmzerolog) for more information about Zerolog integration.


### Slog [_slog]

We support log correlation and error tracking with [Slog](https://pkg.go.dev/log/slog/), [v1.21.0](https://pkg.go.dev/log/slog@go1.21.0/) and greater.

See [module/apmslog](/reference/builtin-modules.md#builtin-modules-apmslog) for more information about slog integration.


## Object Storage [supported-tech-object-storage]


### Amazon S3 [_amazon_s3]

We provide instrumentation for AWS S3. This is usable with [AWS SDK Go](https://github.com/aws/aws-sdk-go).

See [module/apmawssdkgo](/reference/builtin-modules.md#builtin-modules-apmawssdkgo) for more information about AWS SDK Go instrumentation.


### Azure Storage [_azure_storage]

We provide instrumentation for Azure Storage. This is usable with:

* github.com/Azure/azure-storage-blob-go/azblob[Azure Blob Storage]
* github.com/Azure/azure-storage-queue-go/azqueue[Azure Queue Storage]
* github.com/Azure/azure-storage-file-go/azfile[Azure File Storage]

See [module/apmazure](/reference/builtin-modules.md#builtin-modules-apmazure) for more information about Azure SDK Go instrumentation.


## Messaging Systems [supported-tech-messaging-systems]


### Amazon SQS [_amazon_sqs]

We provide instrumentation for AWS SQS. This is usable with [AWS SDK Go](https://github.com/aws/aws-sdk-go).

See [module/apmawssdkgo](/reference/builtin-modules.md#builtin-modules-apmawssdkgo) for more information about AWS SDK Go instrumentation.


### Amazon SNS [_amazon_sns]

We provide instrumentation for AWS SNS. This is usable with [AWS SDK Go](https://github.com/aws/aws-sdk-go).

See [module/apmawssdkgo](/reference/builtin-modules.md#builtin-modules-apmawssdkgo) for more information about AWS SDK Go instrumentation.

