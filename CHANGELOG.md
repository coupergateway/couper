# Couper Changelog

<a name="0.5"></a>
## [0.5](https://github.com/avenga/couper/compare/0.4.2...0.5)

> 2021-01-29

### Bug Fixes

* Fix missing http.Hijacker interface to be able to handle websocket upgrades ([#80](https://github.com/avenga/couper/issues/80))

### Features

* Add additional eval functions: coalesce, json_decode, json_encode ([#105](https://github.com/avenga/couper/pull/105))
* Add multi API support ([#103](https://github.com/avenga/couper/issues/103))
* Add free endpoints ([#90](https://github.com/avenga/couper/issues/90))
* Add remove_, set_ and  add_headers ([#98](https://github.com/avenga/couper/issues/98))

### Code Refactoring

* improved internals for configuration load

### Dependencies

* Upgrade hcl to 2.8.2
* Upgrade go-cty module to 1.5.0
* Upgrade logrus module to 1.7.0
* Upgrade kin-openapi module to v0.37

<a name="0.4.2"></a>
## [0.4.2](https://github.com/avenga/couper/compare/0.4.1...0.4.2)

> 2021-01-19

### Fix

* Fix used backend hash not dependent on (hcl) config hierarchy (transport key)
* Fix logging http scheme even without a successful tls handshake ([#99](https://github.com/avenga/couper/pull/99))
* Fix hcl.Body content for reference backends ([#96](https://github.com/avenga/couper/issues/96))

<a name="0.4.1"></a>
## [0.4.1](https://github.com/avenga/couper/compare/0.4...0.4.1)

> 2021-01-18

### Fix

* Fix path trailing slash ([#94](https://github.com/avenga/couper/issues/94))
* Fix query encoding ([#93](https://github.com/avenga/couper/issues/93))
* Fix log_format (settings) configuration ([#61](https://github.com/avenga/couper/issues/61))

<a name="0.4"></a>
## [0.4](https://github.com/avenga/couper/compare/v0.3...0.4)

> 2021-01-13

### Add

* url log field ([#87](https://github.com/avenga/couper/issues/87))
* Add proxy from env settings option ([#84](https://github.com/avenga/couper/issues/84))
* Add backend settings:  `disable_certificate_validation`, `max_connections` ([#86](https://github.com/avenga/couper/issues/86))

### Fix

* command flag filter for bool values ([#85](https://github.com/avenga/couper/issues/85))
* different proxy options for same origin should be part of the origin transport key

### Refactor

* configuration load and prepare related body merges on hcl level

<a name="v0.3"></a>
## [v0.3](https://github.com/avenga/couper/compare/v0.2...v0.3)

> 2020-12-15

### Add

* build version to startup log
* upstream request/response validation with `openapi` ([#21](https://github.com/avenga/couper/issues/21)) ([#22](https://github.com/avenga/couper/issues/22))
* request-id: uuid v4 format option [#31](https://github.com/avenga/couper/issues/31) ([#53](https://github.com/avenga/couper/issues/53))
* `path_params` [#59](https://github.com/avenga/couper/issues/59)
* gzip support ([#66](https://github.com/avenga/couper/issues/66))
* `query_params` ([#73](https://github.com/avenga/couper/issues/73))
* `json_body` access for request and response bodies [#44](https://github.com/avenga/couper/issues/44) ([#60](https://github.com/avenga/couper/issues/60))

### Change

* start Couper via `run` command now
* internal router [#59](https://github.com/avenga/couper/issues/59)
* docker tag behaviour on release [#70](https://github.com/avenga/couper/issues/70) ([#82](https://github.com/avenga/couper/issues/82))
* request/response_headers to use `set` prefix ([#77](https://github.com/avenga/couper/issues/77))
* passing the filename to underlying hcl diagnostics
* Dockerfile to provide simple file serving ([#63](https://github.com/avenga/couper/issues/63))

### Fix

* handling cty null or unknown values during roundtrip eval [#71](https://github.com/avenga/couper/issues/71)
* logging: start-time measurement
* missing `backend.hostname` documentation ([#62](https://github.com/avenga/couper/issues/62))


<a name="v0.2"></a>
## [v0.2](https://github.com/avenga/couper/compare/v0.1...v0.2)

> 2020-10-08

### Add

* health check ([#29](https://github.com/avenga/couper/issues/29))
* Basic-Auth support ([#19](https://github.com/avenga/couper/issues/19))
* post (form) parsing for use in config variables ([#26](https://github.com/avenga/couper/issues/26))
* more documentation

### Fix

* wildcard path join with trailing slash and respect req path ([#45](https://github.com/avenga/couper/pull/45))
* env var mapping ([#35](https://github.com/avenga/couper/pull/35))
* JWT HMAC keys ([#32](https://github.com/avenga/couper/pull/32))


<a name="0.1"></a>
## 0.1

> 2020-09-11

### Add

* Parse and load from given *HCL* configuration file
* Config structs for blocks: `server, api, endpoint, files, spa, definitions, jwt`
* HTTP handler implementation for `api backends, files, spa` and related config mappings
* CORS handling for `api` endpoints
* Access control configuration for all blocks
* Access control type `jwt` with claim validation
* _Access_ und _backend_ logs
* Configurable error templates with a fallback to our [defaults](./assets/files)
* Github actions for our continuous integration workflows
* [Dockerfile](./Dockerfile)
* [Documentation](./docs)
