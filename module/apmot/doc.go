// Package apmot provides an Elastic APM implementation of the OpenTracing API.
//
// Things not implemented by this tracer:
//  - binary propagation format
//  - baggage
//  - logging
//
// TODO(axw)
//  - investigate injecting native APM transactions/spans
//    as the parent when starting an OT span. This probably
//    requires extending the OT API.
package apmot
