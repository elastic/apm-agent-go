---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/getting-started.html
---

# Set up the APM Go Agent [getting-started]

To start reporting your Go application’s performance to Elastic APM, you need to do a few things:

1. [Install the Agent](#installation).
2. [Instrument Go Source Code](#instrumenting-source).
3. [Configure the Agent](#configure-setup).


## Install the Agent [installation]

Within a Go module, install the Elastic APM Go agent package using `go get`:

```bash
go get -u go.elastic.co/apm/v2
```


### Requirements [_requirements]

You can find a list of the supported frameworks and other technologies in the [*Supported Technologies*](/reference/supported-technologies.md) section.


## Instrument Go Source Code [instrumenting-source]

Instrumentation is the process of extending your application’s code to report trace data to Elastic APM. Go applications must be instrumented manually at the source code level. There are two ways to instrument your applications:

* Using [Built-in instrumentation modules](/reference/builtin-modules.md).
* [Custom instrumentation](/reference/custom-instrumentation.md) and [Context propagation](/reference/custom-instrumentation-propagation.md) with the Go Agent API.

Where possible, use the built-in modules to report transactions served by web and RPC frameworks in your application.


## Configure the Agent [configure-setup]

To simplify development and testing, the agent defaults to sending data to the Elastic APM Server at `http://localhost:8200`. To send data to an alternative location, you must configure [ELASTIC_APM_SERVER_URL](/reference/configuration.md#config-server-url). Depending on the configuration of your server, you may also need to set [ELASTIC_APM_API_KEY](/reference/configuration.md#config-api-key), [ELASTIC_APM_SECRET_TOKEN](/reference/configuration.md#config-secret-token), and [ELASTIC_APM_VERIFY_SERVER_CERT](/reference/configuration.md#config-verify-server-cert). All other variables have usable defaults.

See [*Configuration*](/reference/configuration.md) to learn about all available options.




