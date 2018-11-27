# apmgokit

Package apmgokit provides examples and integration tests
for tracing services implemented with Go kit.

We do not provide any Go kit specific code, as the other
generic modules ([module/apmhttp](../apmhttp) and [module/apmgrpc](../apmgrpc))
are sufficient.

Go kit-based HTTP servers can be traced by instrumenting the
kit/transport/http.Server with apmhttp.Wrap, and HTTP clients
can be traced by providing a net/http.Client instrumented with
apmhttp.WrapClient.

Go kit-based gRPC servers and clients can both be wrapped using
the interceptors provided in module/apmgrpc.
