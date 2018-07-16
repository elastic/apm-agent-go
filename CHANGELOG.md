# Changelog

## [Unreleased](https://github.com/elastic/apm-agent-go/compare/v0.4.0...master)

 - `ELASTIC_APM_TRANSPORT` now defaults to "http://localhost:8200" (#122)
 - `Transport.SetUserAgent` method added, enabling the User-Agent to be set programatically (#124)
 - Inlined functions are now properly reported in stacktraces (#127)
 - Support for the experimental metrics API added (#94)
 - module/apmsql: SQL is parsed to generate more useful span names (#129)
 - Basic vgo module added (#136)
 - module/apmhttprouter: added a wrapper type for httprouter.Router to simplify adding routes (#140)

## [v0.4.0](https://github.com/elastic/apm-agent-go/releases/tag/v0.4.0)

First release of the Go agent for Elastic APM
