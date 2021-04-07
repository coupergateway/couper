# Couper Changelog

<a name="v0.8"></a>
## [v0.8](https://github.com/avenga/couper/compare/0.7.0...v0.8)

> 2021-04-06

### Bug Fixes

* Some possible race conditions in combination with multiple `proxy` and/or `request`
  definitions are fixed ([#157](https://github.com/avenga/couper/issues/177)) ([#160](https://github.com/avenga/couper/issues/160))
* Log endpoint related recovered panics
* CORS behaviour: result is now only dependent on the config, not the actual request; fixed Vary headers ([#173](https://github.com/avenga/couper/issues/173))
* Fix json type assumption ([#177](https://github.com/avenga/couper/issues/177))
  * `req.json_body` result is an empty object for specific types ([#165](https://github.com/avenga/couper/issues/165))
  * Empty json array encodes to `null`. ([#162](https://github.com/avenga/couper/issues/162))
* Fix missing string conversion for evaluated number values ([#175](https://github.com/avenga/couper/issues/175))
* Loading optional labels of same type
* multiplexer behaviour with multiple servers and hosts ([#161](https://github.com/avenga/couper/issues/161))
* Fix missing access_control for file handler ([#169](https://github.com/avenga/couper/issues/169))
* 404 behaviour for access controlled endpoints:
  deny instead of 404 if the request matches the related base_path ([#143](https://github.com/avenga/couper/issues/143))

### Changes

* Rename log type for backend requests: `couper_upstream` -> `couper_backend` ([#159](https://github.com/avenga/couper/pull/159)) ([#172](https://github.com/avenga/couper/pull/172))
* Rename `post` variable to `form_body` ([#158](https://github.com/avenga/couper/pull/158))

### Features

* Add `json_body` attribute for `request` and `response` block ([#158](https://github.com/avenga/couper/issues/158))
* `bytes` log field to represent the body size


<a name="0.7.0"></a>
## [0.7.0](https://github.com/avenga/couper/compare/0.6.1...0.7.0)

> 2021-03-23

### Bug Fixes

* Recover from possible request/proxy related panics ([#157](https://github.com/avenga/couper/pull/157)) ([#145](https://github.com/avenga/couper/pull/145))
* Configuration related hcl merge with an empty attributes and nested blocks

### Change

* `backend` block attributes `basic_auth`, `path_prefix` and `proxy` hcl evaluation during runtime
* `request` attributes hcl evaluation during runtime ([#152](https://github.com/avenga/couper/pull/152))
* Change configuration in combination with URL and backend.origin ([#144](https://github.com/avenga/couper/issues/144))
  * `request` and `proxy` block can use the `url` attribute instead of define or reference a `backend`
  * same applies to `oauth2.token_endpoint`
* no `X-Forwarded-For` header enrichment from couper `proxy` ([#139](https://github.com/avenga/couper/pull/139))
* more log context for access control related errors ([#154](https://github.com/avenga/couper/issues/154))

### Features

* `saml` 2.0 `access_control` support ([#113](https://github.com/avenga/couper/issues/113))
* Add new `strip-secure-cookies` setting ([#147](https://github.com/avenga/couper/issues/147))
  * removes `Secure` flag from all `Set-Cookie` header
* CORS support (`server`, `files`, `spa`) ([#134](https://github.com/avenga/couper/issues/134))
  * previously `api` only
* `error_file` attribute for `endpoint` block
* hcl functions:
  * `merge`
  * `url_encode`
* `backend`
  * OAuth2 support ([#130](https://github.com/avenga/couper/issues/130))
    * grant_type: `client_credentials`
    * `token` memory storage with ttl
  * `path_prefix` attribute ([#138](https://github.com/avenga/couper/issues/138))

<a name="0.6.1"></a>
## [0.6.1](https://github.com/avenga/couper/compare/0.6...0.6.1)

> 2021-03-15

### Bug Fixes

* Fix missing panic recovering for backend roundtrips ([#142](https://github.com/avenga/couper/issues/142))
  * Fix backend `timeout` behaviour
  * Add a more specific error message for proxy body copy errors

### Change

* Couper just passes the `X-Forwarded-For` header if any instead of adding the client remote addr to the list ([#139](https://github.com/avenga/couper/pull/139))

### Features

* `url_encode` function for RFC 3986 string encoding ([#136](https://github.com/avenga/couper/pull/136))

<a name="0.6"></a>
## [0.6](https://github.com/avenga/couper/compare/0.5.1...0.6)

> 2021-03-11

### Breaking change

* `backend` will be consumed by proxy and request as transport configuration now. The previous behaviour
  that `backend` represents a `proxy` functionality is removed. Also the `backend` block must be defined in
  `definitions`, `proxy` or `request`.
  * Config migration, add a `proxy` block:
```hcl
  endpoint "/old" {
  backend = "reference"
  # or
  backend {
    #...
  }
}
# change to:
endpoint "/new" {
  proxy {
    backend = "reference"
  }
  # or
  proxy {
    backend {
      #...
    }
  }
}
```

### Change

* Client-Request and upstream response body buffering by default
* Server shutdown delay and deadline defaults to `0s` now and can be configured via [env](DOCKER.md) if required
* Websocket connection upgrades in combination with `proxy {}` are disabled
  * we will add a proxy option for ws usage later on

### Bug Fixes

* An absolute path resolving for `*_file` configuration attributes ([#120](https://github.com/avenga/couper/pull/120))

### Features

* Endpoint:
  * Add `proxy` block to reverse proxy the client request to the configured `backend`.
  * Add `request` block to send a simple upstream request. [Docs](docs/README.md#request-block)
  * Add `response` block to create a custom client response. [Docs](docs/README.md#response-block)
* Add `jwt_sign()` function to be able to create and sign a token with a `jwt_signing_profile`. [Docs](docs/README.md#functions) ([#112](https://github.com/avenga/couper/issues/112))
* Add `unixtime()` function for the current unix time in seconds ([#124](https://github.com/avenga/couper/issues/124))

### Code Refactoring

* underlying code structure to represent an `endpoint` block with `proxy`, `request` and `response` configuration
* hcl evaluation context as own 'container' with `context.Context` interface
* test cleanups

### Dependencies

* build with go 1.16
* logrus to v1.8.1
* hcl to v2.9.1
* kin-openapi to v.0.49.0


<a name="0.5.1"></a>
## [0.5.1](https://github.com/avenga/couper/compare/0.5...0.5.1)

> 2021-02-16

### Features

* backend:
  * a user-friendly `basic_auth` option
  * backend `proxy` url, `disable_connection_reuse` and `http2` settings ([#108](https://github.com/avenga/couper/pull/108))
* version command

### Change

* KeepAlive `60s` ([#108](https://github.com/avenga/couper/pull/108)), previously `15s`
* Reject requests which hits an endpoint with basic-auth access-control, and the configured password evaluates to an empty string ([#115](https://github.com/avenga/couper/pull/115))

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
