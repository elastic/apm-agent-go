---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/configuration.html
---

# Configuration [configuration]

Adapt the Elastic APM Go agent to your needs with one of the following methods—​listed in descending order of precedence:

1. [APM Agent Configuration via Kibana](docs-content://solutions/observability/apps/apm-agent-central-configuration.md) (supported options are marked with [![dynamic config](images/dynamic-config.svg "") ](#dynamic-configuration))
2. In code, using the [Tracer Config API](/reference/api-documentation.md#tracer-config-api)
3. Environment variables

Configuration defined via Kibana will take precedence over the same configuration defined in code, which takes precedence over environment variables. If configuration is defined via Kibana, and then that is later removed, the agent will revert to configuration defined locally via either the Tracer Config API or environment variables.

To simplify development and testing, the agent defaults to sending data to the Elastic APM Server at `http://localhost:8200`. To send data to an alternative location, you must configure [ELASTIC_APM_SERVER_URL](#config-server-url). Depending on the configuration of your server, you may also need to set [ELASTIC_APM_API_KEY](#config-api-key), [ELASTIC_APM_SECRET_TOKEN](#config-secret-token), and [ELASTIC_APM_VERIFY_SERVER_CERT](#config-verify-server-cert). All other variables have usable defaults.


## Dynamic configuration [dynamic-configuration]

Configuration options marked with the ![dynamic config](../images/dynamic-config.svg "") badge can be changed at runtime when set from a supported source.

The Go Agent supports [Central configuration](docs-content://solutions/observability/apps/apm-agent-central-configuration.md), which allows you to fine-tune certain configurations via the APM app. This feature is enabled in the Agent by default, with [`ELASTIC_APM_CENTRAL_CONFIG`](#config-central-config).


## Configuration formats [_configuration_formats]

Some options require a unit, either duration or size. These need to be provided in a specific format.


### Duration format [_duration_format]

The *duration* format is used for options like timeouts. The unit is provided as a suffix directly after the number, without any whitespace.

**Example:** `5ms`

**Supported units:**

* `ms` (milliseconds)
* `s` (seconds)
* `m` (minutes)


### Size format [_size_format]

The *size* format is used for options such as maximum buffer sizes. The unit is provided as a suffix directly after the number, without any whitespace.

**Example:** `10KB`

**Supported units:**

* B (bytes)
* KB (kilobytes)
* MB (megabytes)
* GB (gigabytes)

::::{note}
We use the power-of-two sizing convention, e.g. 1KB = 1024B.
::::



## `ELASTIC_APM_SERVER_URL` [config-server-url]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_SERVER_URL` | `http://localhost:8200` | `http://localhost:8200` |

The URL for your Elastic APM Server. The Server supports both HTTP and HTTPS. If you use HTTPS, then you may need to configure your client machines so that the server certificate can be verified. You can disable certificate verification with [`ELASTIC_APM_VERIFY_SERVER_CERT`](#config-verify-server-cert).


## `ELASTIC_APM_SERVER_TIMEOUT` [config-server-timeout]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_SERVER_TIMEOUT` | `30s` | `30s` |

The timeout for requests made to your Elastic APM server. When set to zero or a negative value, timeouts will be disabled.


## `ELASTIC_APM_SECRET_TOKEN` [config-secret-token]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_SECRET_TOKEN` |  | "A random string" |

This string is used to ensure that only your agents can send data to your APM server. Both the agents and the APM server have to be configured with the same secret token.

::::{warning}
The secret token is sent as plain-text in every request to the server, so you should also secure your communications using HTTPS. Unless you do so, your secret token could be observed by an attacker.
::::



## `ELASTIC_APM_API_KEY` [config-api-key]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_API_KEY` |  | "A base64-encoded string" |

This base64-encoded string is used to ensure that only your agents can send data to your APM server. The API key must be created using the APM Server [command line tool](docs-content://solutions/observability/apps/api-keys.md).

::::{warning}
The API Key is sent as plain-text in every request to the server, so you should also secure your communications using HTTPS. Unless you do so, your API Key could be observed by an attacker.
::::



## `ELASTIC_APM_SERVICE_NAME` [config-service-name]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_SERVICE_NAME` | Executable name | `my-app` |

The name of your service or application.  This is used to keep all the errors and transactions of your service together and is the primary filter in the Elastic APM user interface.

If you do not specify `ELASTIC_APM_SERVICE_NAME`, the Go agent will use the executable name. e.g. if your executable is called "my-app.exe", then your service will be identified as "my-app".

::::{note}
The service name must conform to this regular expression: `^[a-zA-Z0-9 _-]+$`. In other words: your service name must only contain characters from the ASCII alphabet, numbers, dashes, underscores, and spaces.
::::



## `ELASTIC_APM_SERVICE_VERSION` [config-service-version]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_SERVICE_VERSION` |  | A string indicating the version of the deployed service |

A version string for the currently deployed version of the service. If you don’t version your deployments, the recommended value for this field is the commit identifier of the deployed revision, e.g. the output of `git rev-parse HEAD`.


## `ELASTIC_APM_SERVICE_NODE_NAME` [config-service-node-name]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_SERVICE_NODE_NAME` |  | `my-node-name` |

Optional name used to differentiate between nodes in a service. Must be unique, otherwise data from multiple nodes will be aggregated together.

If you do not specify `ELASTIC_APM_SERVICE_NODE_NAME`, service nodes will be identified using the container ID if available, otherwise the host name.

::::{note}
This feature is fully supported in the APM Server versions >= 7.5.
::::



## `ELASTIC_APM_ENVIRONMENT` [config-environment]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_ENVIRONMENT` |  | `"production"` |

The name of the environment this service is deployed in, e.g. "production" or "staging".

Environments allow you to easily filter data on a global level in the APM app. It’s important to be consistent when naming environments across agents. See [environment selector](docs-content://solutions/observability/apps/filter-application-data.md#apm-filter-your-data-service-environment-filter) in the APM app for more information.

::::{note}
This feature is fully supported in the APM app in Kibana versions >= 7.2. You must use the query bar to filter for a specific environment in versions prior to 7.2.
::::



## `ELASTIC_APM_ACTIVE` [config-active]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_ACTIVE` | true | `false` |

Enable or disable the agent. If set to false, then the Go agent does not send any data to the Elastic APM server, and instrumentation overhead is minimized.


## `ELASTIC_APM_RECORDING` [config-recording]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_RECORDING` | true | `false` |

Enable or disable recording of events. If set to false, then the Go agent does not send any events to the Elastic APM server, and instrumentation overhead is minimized, but the agent will continue to poll the server for configuration changes.


## `ELASTIC_APM_GLOBAL_LABELS` [config-global-labels]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_GLOBAL_LABELS` |  | `dept=engineering,rack=number8` |

Labels are added to all events. The format for labels is: `key=value[,key=value[,...]]`. Any labels set by application via the API will override global labels with the same keys.

This option requires APM Server 7.2 or greater, and will have no effect when using older server versions.


## `ELASTIC_APM_TRANSACTION_IGNORE_URLS` [config-ignore-urls]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_TRANSACTION_IGNORE_URLS` |  | `/heartbeat*, *.jpg` |

A list of patterns to match HTTP requests to ignore. An incoming HTTP request whose request line matches any of the patterns will not be reported as a transaction.

This option supports the wildcard `*`, which matches zero or more characters. Examples: `/foo/*/bar/*/baz*`, `*foo*`. Matching is case insensitive by default. Prefixing a pattern with `(?-i)` makes the matching case sensitive.

::::{note}
This configuration was previously known as `ELASTIC_APM_IGNORE_URLS`, which has been deprecated and will be removed in a future major version of the agent.
::::



## `ELASTIC_APM_SANITIZE_FIELD_NAMES` [config-sanitize-field-names]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_SANITIZE_FIELD_NAMES` | `password, passwd, pwd, secret, *key, *token*, *session*, *credit*, *card*, *auth*, set-cookie, *principal*` | `sekrits` |

A list of patterns to match the names of HTTP headers, cookies, and POST form fields to redact.

This option supports the wildcard `*`, which matches zero or more characters. Examples: `/foo/*/bar/*/baz*`, `*foo*`. Matching is case insensitive by default. Prefixing a pattern with `(?-i)` makes the matching case sensitive.


## `ELASTIC_APM_CAPTURE_HEADERS` [config-capture-headers]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_CAPTURE_HEADERS` | `true` |

For transactions that are HTTP requests, the Go agent can optionally capture request and response headers.

Possible values: `true`, `false`.

Captured headers are subject to sanitization, per [`ELASTIC_APM_SANITIZE_FIELD_NAMES`](#config-sanitize-field-names).


## `ELASTIC_APM_CAPTURE_BODY` [config-capture-body]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_CAPTURE_BODY` | `off` |

For transactions that are HTTP requests, the Go agent can optionally capture the request body.

Possible values: `errors`, `transactions`, `all`, `off`.

::::{warning}
Request bodies often contain sensitive values like passwords, credit card numbers, and so on. If your service handles data like this, enable this feature with care.
::::



## `ELASTIC_APM_HOSTNAME` [config-hostname]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_HOSTNAME` | `os.Hostname()` | `app-server01` |

The host name to use when sending error and transaction data to the APM server.


## `ELASTIC_APM_API_REQUEST_TIME` [config-api-request-time]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_API_REQUEST_TIME` | `10s` |

The amount of time to wait before ending a request to the Elastic APM server. When you report transactions, spans and errors, the agent will initiate a request and send them to the server when there is enough data to send; the request will remain open until this time has been exceeded, or until the [maximum request size](#config-api-request-size) has been reached.


## `ELASTIC_APM_API_REQUEST_SIZE` [config-api-request-size]

| Environment | Default | Minimum | Maximum |
| --- | --- | --- | --- |
| `ELASTIC_APM_API_REQUEST_SIZE` | `750KB` | `1KB` | `5MB` |

The maximum size of request bodies to send to the Elastic APM server. The agent will maintain an in-memory buffer of compressed data for streaming to the APM server.


## `ELASTIC_APM_API_BUFFER_SIZE` [config-api-buffer-size]

| Environment | Default | Minimum | Maximum |
| --- | --- | --- | --- |
| `ELASTIC_APM_API_BUFFER_SIZE` | `1MB` | `10KB` | `100MB` |

The maximum number of bytes of uncompressed, encoded events to store in memory while the agent is busy. When the agent is able to, it will transfer buffered data to the request buffer, and start streaming it to the server. If the buffer fills up, new events will start replacing older ones.


## `ELASTIC_APM_TRANSACTION_MAX_SPANS` [config-transaction-max-spans]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_TRANSACTION_MAX_SPANS` | `500` |

Limits the amount of spans that are recorded per transaction.

This is helpful in cases where a transaction creates a large number of spans (e.g. thousands of SQL queries). Setting an upper limit will prevent overloading the agent and the APM server with too much work for such edge cases.


## `ELASTIC_APM_EXIT_SPAN_MIN_DURATION` [config-exit-span-min-duration]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_EXIT_SPAN_MIN_DURATION` | `1ms` |

Sets the minimum duration for an exit span to be reported. Spans shorter or equal to this threshold will be dropped by the agent and reported as statistics in the span’s transaction, as long as the transaction didn’t end before the span was reported.

When span compression is enabled ([`ELASTIC_APM_SPAN_COMPRESSION_ENABLED`](#config-span-compression-enabled)), the sum of the compressed span composite is considered.

The minimum duration allowed for this setting is 1 microsecond (`us`).


## `ELASTIC_APM_SPAN_FRAMES_MIN_DURATION` [config-span-frames-min-duration-ms]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_SPAN_FRAMES_MIN_DURATION` | `5ms` |

The APM agent will collect a stack trace for every recorded span whose duration exceeds this configured value. While this is very helpful to find the exact place in your code that causes the span, collecting this stack trace does have some processing and storage overhead.

::::{note}
This configuration has been deprecated and will be removed in a future major version of the agent.
::::



## `ELASTIC_APM_SPAN_STACK_TRACE_MIN_DURATION` [config-span-stack-trace-min-duration]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_SPAN_STACK_TRACE_MIN_DURATION` | `5ms` |

The APM agent will collect a stack trace for every recorded span whose duration exceeds this configured value. While this is very helpful to find the exact place in your code that causes the span, collecting this stack trace does have some processing and storage overhead.

::::{note}
This configuration was previously known as `ELASTIC_APM_SPAN_FRAMES_MIN_DURATION`, which has been deprecated and will be removed in a future major version of the agent.
::::



## `ELASTIC_APM_STACK_TRACE_LIMIT` [config-stack-trace-limit]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_STACK_TRACE_LIMIT` | `50` |

Limits the number of frames captured for each stack trace.

Setting the limit to 0 will disable stack trace collection, while any positive integer value will be used as the maximum number of frames to collect. Setting a negative value, such as -1, means that all frames will be collected.


## `ELASTIC_APM_TRANSACTION_SAMPLE_RATE` [config-transaction-sample-rate]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_TRANSACTION_SAMPLE_RATE` | `1.0` |

By default, the agent will sample every transaction (e.g. request to your service). To reduce overhead and storage requirements, set the sample rate to a value between `0.0` and `1.0`. We still record overall time and the result for unsampled transactions, but no context information, tags, or spans.


## `ELASTIC_APM_METRICS_INTERVAL` [config-metrics-interval]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_METRICS_INTERVAL` | 30s |

The interval at which APM agent gathers and reports metrics. Set to `0s` to disable.


## `ELASTIC_APM_DISABLE_METRICS` [config-disable-metrics]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_DISABLE_METRICS` |  | `system.*, *cpu*` |

Disables the collection of certain metrics. If the name of a metric matches any of the wildcard expressions, it will not be collected.

This option supports the wildcard `*`, which matches zero or more characters. Examples: `/foo/*/bar/*/baz*`, `*foo*`. Matching is case insensitive by default. Prefixing a pattern with `(?-i)` makes the matching case sensitive.


## `ELASTIC_APM_BREAKDOWN_METRICS` [config-breakdown-metrics]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_BREAKDOWN_METRICS` | `true` |

Capture breakdown metrics. Set to `false` to disable.


## `ELASTIC_APM_SERVER_CERT` [config-server-cert]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_SERVER_CERT` |  |

If you have configured your APM Server with a self signed TLS certificate, or you want to pin the server certificate, specify the path to the PEM-encoded certificate via the `ELASTIC_APM_SERVER_CERT` configuration.


## `ELASTIC_APM_SERVER_CA_CERT_FILE` [config-server-ca-cert-file]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_SERVER_CA_CERT_FILE` |  |

The path to a PEM-encoded TLS Certificate Authority certificate that will be used for verifying the server’s TLS certificate chain.


## `ELASTIC_APM_VERIFY_SERVER_CERT` [config-verify-server-cert]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_VERIFY_SERVER_CERT` | `true` |

By default, the agent verifies the server’s certificate if you use an HTTPS connection to the APM server. Verification can be disabled by changing this setting to `false`. This setting is ignored when `ELASTIC_APM_SERVER_CERT` is set.


## `ELASTIC_APM_LOG_FILE` [config-log-file]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_LOG_FILE` |  |

`ELASTIC_APM_LOG_FILE` specifies the output file for the agent’s default, internal logger. The file will be created, or truncated if it exists, when the process starts. By default, logging is disabled. You must specify `ELASTIC_APM_LOG_FILE` to enable it. This environment variable will be ignored if a logger is configured programatically.

There are two special file names that the agent recognizes: `stdout` and `stderr`. These will configure the logger to write to standard output and standard error respectively.


## `ELASTIC_APM_LOG_LEVEL` [config-log-level]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_LOG_LEVEL` | `"error"` |

`ELASTIC_APM_LOG_LEVEL` specifies the log level for the agent’s default, internal logger. The only two levels used by the logger are "error" and "debug". By default, logging is disabled. You must specify `ELASTIC_APM_LOG_FILE` to enable it.

This environment variable will be ignored if a logger is configured programatically.


### `ELASTIC_APM_CENTRAL_CONFIG` [config-central-config]

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_CENTRAL_CONFIG` | `true` |

Activate APM Agent central configuration via Kibana. By default the agent will poll the server for agent configuration changes. This can be disabled by changing the setting to `false`. See [APM Agent central configuration](docs-content://solutions/observability/apps/apm-agent-central-configuration.md) for more information.

::::{note}
This feature requires APM Server v7.3 or later.
::::



### `ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER` [config-use-elastic-traceparent-header]

|     |     |
| --- | --- |
| Environment | Default |
| `ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER` | `true` |

To enable [distributed tracing](docs-content://solutions/observability/apps/traces.md), the agent adds trace context headers to outgoing HTTP requests made with [module/apmhttp](/reference/builtin-modules.md#builtin-modules-apmhttp). These headers (`traceparent` and `tracestate`) are defined in the [W3C Trace Context](https://www.w3.org/TR/trace-context-1/) specification.

When this setting is `true`, the agent will also add the header `elastic-apm-traceparent` for backwards compatibility with older versions of Elastic APM agents.


### `ELASTIC_APM_CLOUD_PROVIDER` [config-cloud-provider]

| Environment | Default | Example |
| --- | --- | --- |
| `ELASTIC_APM_CLOUD_PROVIDER` | `"auto"` | `"aws"` |

This config value allows you to specify which cloud provider should be assumed for metadata collection. By default, the agent will use trial and error to automatically collect the cloud metadata.

Valid options are `"none"`, `"auto"`, `"aws"`, `"gcp"`, and `"azure"` If this config value is set to `"none"`, then no cloud metadata will be collected.


## `ELASTIC_APM_SPAN_COMPRESSION_ENABLED` [config-span-compression-enabled]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_SPAN_COMPRESSION_ENABLED` | `true` |

When enabled, the agent will attempt to compress *short* exit spans that share the same parent into a composite span. The exact duration for what is considered *short*, depends on the compression strategy used (`same_kind` or `exact_match`).

In order for a span to be compressible, these conditions need to be met:

* Spans are exit spans.
* Spans are siblings (share the same parent).
* Spans have not propagated their context downstream.
* Each span duration is equal or lower to the compression strategy maximum duration.
* Spans are compressed with `same_kind` strategy when these attributes are equal:

    * `span.type`.
    * `span.subtype`.
    * `span.context.destination.service.resource`

* Spans are compressed with `exact_match` strategy when all the previous conditions are met and the `span.name` is equal.

Compressing short exit spans should provide some storage savings for services that create a lot of consecutive short exit spans to for example databases or cache services which are generally uninteresting when viewing a trace.

::::{warning}
This feature is experimental and requires APM Server v7.15 or later.
::::



## `ELASTIC_APM_SPAN_COMPRESSION_EXACT_MATCH_MAX_DURATION` [config-span-compression-exact-match-duration]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_SPAN_COMPRESSION_EXACT_MATCH_MAX_DURATION` | `50ms` |

The maximum duration to consider for compressing sibling exit spans that are an exact match for compression.


## `ELASTIC_APM_SPAN_COMPRESSION_SAME_KIND_MAX_DURATION` [config-span-compression-same-kind-duration]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_SPAN_COMPRESSION_SAME_KIND_MAX_DURATION` | `0ms` |

The maximum duration to consider for compressing sibling exit spans that are of the same kind for compression.


## `ELASTIC_APM_TRACE_CONTINUATION_STRATEGY` [config-trace-continuation-strategy]

[![dynamic config](../images/dynamic-config.svg "") ](#dynamic-configuration)

| Environment | Default |
| --- | --- |
| `ELASTIC_APM_TRACE_CONTINUATION_STRATEGY` | `continue` |

This option allows some control over how the APM agent handles W3C trace-context headers on incoming requests. By default, the traceparent and tracestate headers are used per W3C spec for distributed tracing. However, in certain cases it can be helpful to not use the incoming traceparent header. Some example use cases:

* An Elastic-monitored service is receiving requests with traceparent headers from unmonitored services.
* An Elastic-monitored service is publicly exposed, and does not want tracing data (trace-ids, sampling decisions) to possibly be spoofed by user requests.

Valid options are `continue`, `restart`, and `restart_external`:

continue
:   The default behavior. An incoming `traceparent` value is used to continue the trace and determine the sampling decision.

restart
:   Always ignores the `traceparent` header of incoming requests. A new trace-id will be generated and the sampling decision will be made based on `transaction_sample_rate`. A span link will be made to the incoming `traceparent`.

restart_external
:   If an incoming request includes the `es` vendor flag in `tracestate`, then any `traceparent` will be considered internal and will be handled as described for **continue** above. Otherwise, any `traceparent` is considered external and will be handled as described for **restart** above.

Starting with Elastic Observability 8.2, span links are visible in trace views.

