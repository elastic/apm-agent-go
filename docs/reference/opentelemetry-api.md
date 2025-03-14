---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/opentelemetry.html
---

# OpenTelemetry API [opentelemetry]

The Elastic APM Go Agent provides wrappers to interact with the [OpenTelemetry API](https://opentelemetry.io/).

Traces and metrics created through the OpenTelemetry API will be translated to their Elastic APM equivalent.


## Initializing the tracing bridge [_initializing_the_tracing_bridge]

The OpenTelemetry Tracing bridge is implemented as a Tracer Provider. Once setup, any span created through that provider will be set on the Elastic agent. And the provider will try to find any existing transaction within the context.

Note: our tracer provider bridge is an incomplete implementation of an OpenTelemetry SDK. It is a good solution meant to help migrate from our agent to OpenTelemetry, but shouldn’t be considered as a long-term solution.

To setup this tracer provider, you first need to import the `apmotel` package.

```go
import (
	"go.elastic.co/apm/module/apmotel/v2"
)
```

The apmotel package exposes a `NewTracerProvider` method, which returns an implementation of an OpenTelemetry Tracer Provider.

```go
provider, err := apmotel.NewTracerProvider()
```

By default, the tracer provider will find the default apm tracer. But you can specify a custom one with the `apmotel.WithAPMTracer` argument option.

Once you have obtained this provider, you can configure it as your OpenTelemetry SDK implementation, and use the APIs normally.

```go
import (
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/module/apmotel/v2"

	"go.opentelemetry.io/otel"
)

func main() {
	provider, err := apmotel.NewTracerProvider()
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(provider)

	tracer := otel.GetTracerProvider().Tracer("example")
	// Start a new span to track some work, which will be sent to the Elastic APM tracer
	ctx, span := tracer.Start(context.Background(), "my_work")
	// Do something
	span.End()
}
```


## Initializing the metrics exporter [opentelemetry-metrics-init]

::::{warning}
The Metrics Exporter is in technical preview until OpenTelemetry marks them as GA.
::::


The OpenTelemetry Metrics bridge is implemented as a manual reader exporter. The Elastic APM Agent will regularly ask OpenTelemetry for its latest metrics, and emit them as its own.

To initialize this exporter, you first need to import the `apmotel` package.

```go
import (
	"go.elastic.co/apm/module/apmotel"
)
```

The apmotel package exposes a `NewGatherer` method, which returns an implementation of both an [Elastic MetricsGatherer](https://pkg.go.dev/github.com/elastic/apm-agent-go#MetricsGatherer), and an [OpenTelemetry metric.Reader](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric#Reader).

```go
exporter := apmotel.NewGatherer()
```

The method allows passing some options, such as `WithAggregationSelector`, to specify a custom OpenTelemetry aggregation selector.

Once you have obtained this exporter, you can configure the OpenTelemetry SDK so it reports emitted metrics, as well as the Elastic Agent to read those metrics.

```go
import (
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmotel"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	exporter := apmotel.NewGatherer()

	// Configure OpenTelemetry
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	// Configure the Elastic APM
	apm.DefaultTracer().RegisterMetricsGatherer(exporter)

	// Record a metric with OpenTelemetry which will be exported to the Elastic Agent
	meter := provider.Meter("my_application")
	counter, _ := meter.Float64Counter("metric_called")
	counter.Add(context.TODO(), 1)
}
```

