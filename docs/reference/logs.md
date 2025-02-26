---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/logs.html
---

# Logs [logs]

Elastic APM Go Agent provides [*Log Correlation*](/reference/log-correlation.md). The agent will automaticaly inject correlation IDs that allow navigation between logs, traces and services.

This features is part of [Application log ingestion strategies](docs-content://solutions/observability/logs/stream-application-logs.md).

The [`ecslogrus`](ecs-logging-go-logrus://docs/reference/index.md) and [`ecszap`](ecs-logging-go-zap://docs/reference/index.md) libraries can also be used to use the [ECS logging](ecs-logging://docs/reference/intro.md) format without an APM agent. When deployed with the Go APM agent, the agent will provide [log correlation](/reference/log-correlation.md) IDs.

The Go agent provides integrations for popular logging frameworks that inject trace ID fields into the application’s log records. You can find a list of the supported logging frameworks under [supported technologies](/reference/supported-technologies.md#supported-tech-logging).

If your favorite logging framework is not already supported, there are two other options:

* Open a feature request, or contribute code, for additional support as described in [*Contributing*](/reference/contributing.md).
* Manually inject trace IDs into log records, as described below in [Manual log correlation](/reference/log-correlation.md#log-correlation-manual).

