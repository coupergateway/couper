# Couper Changelog

## [Unreleased](https://github.com/coupergateway/couper/compare/v1.12.2...master)

Unreleased changes are available as `coupergateway/couper:edge` container.

* **Added**
  * [`can()` function](https://docs.couper.io/configuration/functions) ([#699](https://github.com/coupergateway/couper/pull/699))
  * [`url_decode()` function](https://docs.couper.io/configuration/functions) ([#781](https://github.com/coupergateway/couper/pull/781))
  * `bearer = true` attribute for [`jwt` block](https://docs.couper.io/configuration/block/jwt#attributes) to indicate retrieving token from `Authorization: Bearer ...`. This is the new default token source indicator. `header = "Authorization"` is now _deprecated_ in favour of this new attribute. ([#724](https://github.com/coupergateway/couper/pull/724))
  * IPv6 support via [`-bind-address`](https://docs.couper.io/configuration/command-line#network-options) option. ([#752](https://github.com/coupergateway/couper/pull/752))
  * also watch files which has been referenced within the configuration file when using [`-watch`] (https://docs.couper.io/configuration/command-line#basic-options) ([#747](https://github.com/coupergateway/couper/pull/747))
  * automatic [`MAXPROCS`](https://pkg.go.dev/runtime#GOMAXPROCS) setting for Couper runtime to respect the number of available CPU resources in cloud environments ([#840](https://github.com/coupergateway/couper/pull/840))

* **Changed**
  * More specific error log messages for [`oauth2`](https://docs.couper.io/configuration/block/oauth2) and [`beta_token_request`](https://docs.couper.io/configuration/block/token_request) token request errors ([#755](https://github.com/coupergateway/couper/pull/755))
  * In addition to having an appropriate JSON media type in the `Content-Type` header field, (backend) requests or backend responses for an endpoint are only JSON-parsed if indicated by a [`.json_body` reference](https://docs.couper.io/configuration/variables) in the endpoint configuration ([#749](https://github.com/coupergateway/couper/pull/749))
  * `beta_rate_limit` status code `429` responses are no longer wrapped as a Couper error ([#827](https://github.com/coupergateway/couper/issues/827))

* **Fixed**
  * `WWW-Authenticate` header `realm` param value for [`basic_auth`](https://docs.couper.io/configuration/block/basic_auth) ([#715](https://github.com/coupergateway/couper/pull/715))
  * [`Server-Timing` header](https://docs.couper.io/configuration/block/settings) only reporting last requests/proxies of [endpoint sequences](https://docs.couper.io/configuration/block/endpoint#endpoint-sequence) ([#751](https://github.com/coupergateway/couper/pull/751))
  * Selecting of appropriate [error handler](https://docs.couper.io/configuration/block/error_handler) in two cases ([#753](https://github.com/coupergateway/couper/pull/753))
  * Storing of digit-starting string object keys in [request context](https://docs.couper.io/configuration/variables#request) and of digit-starting string header field names in [request](https://docs.couper.io/configuration/variables#request) variable ([#799](https://github.com/coupergateway/couper/pull/799))
  * Use of boolean values for the `headers` attribute or [modifiers](https://docs.couper.io/configuration/modifiers) ([#805](https://github.com/coupergateway/couper/pull/805))
  * Duplicate [CORS](https://docs.couper.io/configuration/block/cors) response headers (with backend sending CORS response headers, too) ([#804](https://github.com/coupergateway/couper/pull/804))
  * Erroneously sending 404 when serving from [`files`](https://docs.couper.io/configuration/block/files) due to wrong registration of `base_path`s, and when serving from multiple `files` or [`spa`](https://docs.couper.io/configuration/block/spa) in combination with [`api`](https://docs.couper.io/configuration/block/api) due to wrong selecting of the API error template ([#803](https://github.com/coupergateway/couper/pull/803))
  * Possible deadlock for `beta_rate_limit` ([#827](https://github.com/coupergateway/couper/issues/827))

* **Dependencies**
  * build with go 1.22 ([#810](https://github.com/coupergateway/couper/pull/810))
  * upgrade jwt library from v4 to v5 ([#769](https://github.com/coupergateway/couper/issues/769), [#834](https://github.com/coupergateway/couper/pull/834))
  * update OpenAPI lib to v0.126.0 ([#837](https://github.com/coupergateway/couper/pull/837))

---

## [1.12.2](https://github.com/coupergateway/couper/releases/tag/v1.12.2)

* **Fixed**
  * Reading the origin response-body even if there is no origin `body` or `json_body` variable reference; piping the response-body again to the client ([#766](https://github.com/coupergateway/couper/issues/766))

## [1.12.1](https://github.com/coupergateway/couper/releases/tag/v1.12.1)

* **Fixed**
  * Erroneously sending an empty [`Server-Timing` header](https://docs.couper.io/configuration/command-line#oberservation-options) ([#700](https://github.com/coupergateway/couper/pull/700))
  * URL scheme while using the [`tls` block](https://docs.couper.io/configuration/block/server_tls) ([#703](https://github.com/coupergateway/couper/issues/703))
  * For [OIDC](https://docs.couper.io/configuration/block/oidc), trying to request userinfo from a non-existing (not required, though recommended) userinfo endpoint ([#709](https://github.com/coupergateway/couper/pull/709))
  * Use of [`backend_responses`'](https://docs.couper.io/configuration/variables#backend_responses) `body` or `json_body` properties in api-level [error handlers](https://docs.couper.io/configuration/block/error_handler) ([#710](https://github.com/coupergateway/couper/pull/710))
  * Some `..._file` attributes missing for path absolutizing ([#713](https://github.com/coupergateway/couper/pull/713))
  * `WWW-Authenticate` header `realm` param value for [`basic_auth`](https://docs.couper.io/configuration/block/basic_auth) ([#715](https://github.com/coupergateway/couper/pull/715))
  * [JWT access control](https://docs.couper.io/configuration/block/jwt) now creating `401` error status code, adding a `WWW-Authenticate: Bearer[...]` response header if appropriate ([#719](https://github.com/coupergateway/couper/pull/719))
  * Erroneous multiplying of [health probes](https://docs.couper.io/configuration/block/health), [jobs](https://docs.couper.io/configuration/block/job) and requests to [JWKS](https://docs.couper.io/configuration/block/jwt) and [OpenID configuration](https://docs.couper.io/configuration/block/oidc) resources after a reload with [`-watch`](https://docs.couper.io/configuration/command-line#basic-options) ([#730](https://github.com/coupergateway/couper/pull/730), [#736](https://github.com/coupergateway/couper/pull/736))
  * Reading PEM-encoded CA certificates ([`ca_file` setting](https://docs.couper.io/configuration/block/settings#attributes) or [`-ca-file` option](https://docs.couper.io/configuration/command-line#tls-options)) containing bytes trailing the PEM message ([#739](https://github.com/coupergateway/couper/pull/739))

## [1.12.0](https://github.com/coupergateway/couper/releases/tag/v1.12.0)

* **Added**
  * [`beta_job`](https://docs.couper.io/configuration/block/job) block to describe one or more job `definitions` for simple recurring http tasks ([#610](https://github.com/coupergateway/couper/pull/610))
  * [`server_timing_header`](https://docs.couper.io/configuration/block/settings) setting, that allows Couper to include an additional `Server-Timing` HTTP response header field detailing connection and transport relevant metrics for each backend request. ([#657](https://github.com/coupergateway/couper/pull/657))

* **Changed**
  * Use nested `jwt_signing_profile` block in [`oauth2` block](https://docs.couper.io/configuration/block/oauth2) for `grant_type` `"urn:ietf:params:oauth:grant-type:jwt-bearer"` in absence of `assertion` attribute ([#619](https://github.com/coupergateway/couper/pull/619))
  * Improved the way an SPA `bootstrap_file` gets cached and served in combination with `bootstrap_data` ([#656](https://github.com/coupergateway/couper/pull/656))
  * Harmonized and improved logged error information for references to undefined blocks ([#651](https://github.com/coupergateway/couper/pull/651))
  * Unbeta permission features: ([#673](https://github.com/coupergateway/couper/pull/673))
    * `beta_required_permission` attribute for [`api`](https://docs.couper.io/configuration/block/api#attribute-beta_required_permission) and [`endpoint`](https://docs.couper.io/configuration/block/endpoint#attribute-beta_required_permission) blocks,
    * `beta_granted_permissions` and `beta_required_permission` [request context variables](https://docs.couper.io/configuration/variables#request),
    * `beta_insufficient_permissions` [error type](https://docs.couper.io/configuration/error-handling/#api-error-types),
    * `beta_permissions_claim`, `beta_permissions_map`, `beta_permissions_map_file`, `beta_roles_claim`, `beta_roles_map` and `beta_roles_map_file` attributes for [`jwt` block](https://docs.couper.io/configuration/block/jwt#attributes).

* **Fixed**
  * Use of [backend-related variables](https://docs.couper.io/configuration/variables#backend) in [`custom_log_fields`](https://docs.couper.io/observation/logging#custom-logging) within a [`backend` block](https://docs.couper.io/configuration/block/backend) ([#658](https://github.com/coupergateway/couper/pull/658))
  * Loop with evaluation error in [`custom_log_fields`](https://docs.couper.io/observation/logging#custom-logging) if log level is `"debug"` ([#659](https://github.com/coupergateway/couper/pull/659))
  * Removed error message with `couper help` [command](https://docs.couper.io/configuration/command-line) ([#678](https://github.com/coupergateway/couper/pull/678))

## [1.11.2](https://github.com/coupergateway/couper/releases/tag/v1.11.2)

* **Fixed**
  * Requests to wildcard (`**`) [endpoints](https://docs.couper.io/configuration/block/endpoint) using backends with a wildcard [`path` attribue](https://docs.couper.io/configuration/block/backend#attributes), where the wildcard matches the empty string (regression; since v1.11.0) ([#655](https://github.com/coupergateway/couper/pull/655))
  * [internal alloc] stop creating request context based jwt, oauth2 and saml (hcl) functions without related definitions ([#666](https://github.com/coupergateway/couper/pull/666))
  * [internal alloc] reduced allocation amount while proxying requests ([#666](https://github.com/coupergateway/couper/pull/666))
  * Removing websockets related headers while the proxy `websockets` option is `false` (or no block definition exist) ([#666](https://github.com/coupergateway/couper/pull/666))

## [1.11.1](https://github.com/coupergateway/couper/releases/tag/v1.11.1)

* **Fixed**
  * [Endpoint sequences](https://docs.couper.io/configuration/block/endpoint#endpoint-sequence) not being terminated by errors (e.g. `unexpected_status`) (regression; since v1.11.0) ([#648](https://github.com/coupergateway/couper/pull/648))
  * [Health route](https://docs.couper.io/observation/health) affected by [access control](https://docs.couper.io/configuration/access-control) (regression; since v1.11.0) ([#654](https://github.com/coupergateway/couper/pull/654))


## [1.11.0](https://github.com/coupergateway/couper/releases/tag/v1.11.0)

* **Added**
  * **mTLS** Support for [`server`](https://docs.couper.io/configuration/block/server_tls) and [`backend`](https://docs.couper.io/configuration/block/backend_tls) blocks ([#615](https://github.com/coupergateway/couper/pull/615))
  * `spa` block option to inject server-data to the applications `bootstrap_file` with [`bootstrap_data`](https://docs.couper.io/configuration/block/spa#bootstrap-data) ([#626](https://github.com/coupergateway/couper/issues/626))
  * OAuth2 client authentication methods (`token_endpoint_auth_method` values) `"client_secret_jwt"` and `"private_key_jwt"` including `jwt_signing_profile` block for [`oauth2`](https://docs.couper.io/configuration/block/oauth2_req_auth), [`beta_oauth2`](https://docs.couper.io/configuration/block/oauth2_ac) and [`oidc`](https://docs.couper.io/configuration/block/oidc) blocks ([#599](https://github.com/coupergateway/couper/pull/599))
  * [`trim()` function](https://docs.couper.io/configuration/functions) ([#605](https://github.com/coupergateway/couper/pull/605))
  * `beta_roles_map_file` and `beta_permissions_map_file` attributes to [`jwt` block](https://docs.couper.io/configuration/block/jwt) ([#613](https://github.com/coupergateway/couper/pull/613))

* **Changed**
  * Replaced the JWT library because the former library was no longer maintained ([#612](https://github.com/coupergateway/couper/pull/612))
  * Routing and [OpenAPI validation](https://docs.couper.io/configuration/block/openapi) now use gorilla/mux ([#614](https://github.com/coupergateway/couper/pull/614))
  * Usage of `env` variables and functions is now possible for the `defaults` block ([#630](https://github.com/coupergateway/couper/pull/630))

* **Fixed**
  * Aligned the evaluation of [`beta_oauth2`](https://docs.couper.io/configuration/block/oauth2_ac)/[`oidc`](https://docs.couper.io/configuration/block/oidc) `redirect_uri` to `saml` `sp_acs_url` ([#589](https://github.com/coupergateway/couper/pull/589))
  * Proper handling of empty [`beta_oauth2`](https://docs.couper.io/configuration/block/oauth2_ac)/[`oidc`](https://docs.couper.io/configuration/block/oidc) `scope` ([#593](https://github.com/coupergateway/couper/pull/593))
  * Throwing [sequence errors](https://docs.couper.io/configuration/error-handling#endpoint-error-types) and selecting appropriate [error handlers](https://docs.couper.io/configuration/error-handling) ([#595](https://github.com/coupergateway/couper/pull/595))
  * Allow setting of the `typ` JWT header in [`jwt_signing_profile`s](https://docs.couper.io/configuration/block/jwt_signing_profile) ([#616](https://github.com/coupergateway/couper/pull/616))
  * CVE-2021-3538 related to our `request_id_format` option if switched to `uuid4`: replaced the underlying package to `github.com/google/uuid` ([#611](https://github.com/coupergateway/couper/pull/611))
  * Possible panic for nested [endpoint sequences](https://docs.couper.io/configuration/block/endpoint#endpoint-sequence) ([#618](https://github.com/coupergateway/couper/pull/618))
  * Cycle check for [endpoint sequences](https://docs.couper.io/configuration/block/endpoint#endpoint-sequence) ([#623](https://github.com/coupergateway/couper/pull/623))
  * In [endpoint sequences](https://docs.couper.io/configuration/block/endpoint#endpoint-sequence) send requests only once ([#624](https://github.com/coupergateway/couper/pull/624))


## [1.10.1](https://github.com/coupergateway/couper/releases/tag/v1.10.1)

* **Fixed**
  * [endpoint `/**` path wildcards](https://docs.couper.io/configuration/block/endpoint) sometimes not matching ([#603](https://github.com/coupergateway/couper/pull/603))
  * Some errors in the [`default()` function](https://docs.couper.io/configuration/functions) ([#596](https://github.com/coupergateway/couper/pull/596))

## [1.10.0](https://github.com/coupergateway/couper/releases/tag/v1.10.0)

Couper 1.10 is a feature release comprising new features for a more flexible and cleaner configuration. It also comes with some bug fixes and some smaller improvements. See below for a complete list of changes.

The new [`environment` block](https://docs.couper.io/configuration/block/environment) along with its corresponding [setting](https://docs.couper.io/configuration/block/settings) and the [`-e` command line option](https://docs.couper.io/configuration/command-line#basic-options) allows for a cleaner and safer configuration if Couper is to be deployed in different environments. Read more about that feature in our [example](https://github.com/coupergateway/couper-examples/tree/master/environment).

Labeled, reusable [`proxy` blocks](https://docs.couper.io/configuration/block/proxy) provide means for a leaner and less repetitive configuration.

In case a backend request requires authorization, the new [`beta_token_request` block](https://docs.couper.io/configuration/block/token_request) can request the required token and make the response available via the [`beta_token_response` variable](https://docs.couper.io/configuration/variables#beta_token_response).

Finally, a stricter [`endpoint`](https://docs.couper.io/configuration/block/endpoint) path validation for a clearer and more consistent path matching is now applied. Some characters or character sequences like `?`, `#`, and `/../` are no longer allowed; empty path parameters won't match anymore.

As always the [Couper VSCode extension](https://marketplace.visualstudio.com/items?itemName=couper.couperconf) has been updated to support all new features.

We have launched our [documentation website](https://docs.couper.io/) to find blocks and attributes more easily.

* **Added**
  * [`environment` block](https://docs.couper.io/configuration/block/environment), [setting](https://docs.couper.io/configuration/block/settings) and [`couper.environment` variable](https://docs.couper.io/configuration/variables#couper) ([#521](https://github.com/coupergateway/couper/pull/521), ([#534](https://github.com/coupergateway/couper/pull/534), [#545](https://github.com/coupergateway/couper/pull/545)), ([#546](https://github.com/coupergateway/couper/pull/546))
  * used go version in `version` command ([#552](https://github.com/coupergateway/couper/pull/552))
  * new `grant_type`s `"password"` and `"urn:ietf:params:oauth:grant-type:jwt-bearer"` with related attributes for [`oauth2` block](https://docs.couper.io/configuration/block/oauth2) ([#555](https://github.com/coupergateway/couper/pull/555))
  * [`beta_token_request` block](https://docs.couper.io/configuration/block/token_request), [`backend`](https://docs.couper.io/configuration/variables#backend) and [`beta_token_response`](https://docs.couper.io/configuration/variables#beta_token_response) variables and `beta_token(s)` properties of [`backends` variable](https://docs.couper.io/configuration/variables#backends) ([#517](https://github.com/coupergateway/couper/pull/517))
  * reusable [`proxy` block](https://docs.couper.io/configuration/block/proxy) ([#561](https://github.com/coupergateway/couper/pull/561))

* **Changed**
 * Renamed `-debug` to `-pprof` and made debug port configurable via `-pprof-port`. Both [command line options](https://docs.couper.io/configuration/command-line#profiling-options) can also be specified via the respective [`settings`](https://docs.couper.io/configuration/block/settings). ([#577](https://github.com/coupergateway/couper/pull/577))

* **Fixed**
  * [`form_body`, `headers` and `cookies`](https://docs.couper.io/configuration/block/request#attributes) can now be properly [custom-logged](https://docs.couper.io/observation/logging#custom-logging) ([#535](https://github.com/coupergateway/couper/pull/535))
  * Disallow empty path parameters ([#526](https://github.com/coupergateway/couper/pull/526))
  * Disallow endpoint path patterns not starting with `/`, endpoint path patterns and `base_path`s having `.` or `..` segments ([#584](https://github.com/coupergateway/couper/pull/584))
  * Basic Auth client authentication with OAuth2 (client ID and secret must be URL encoded) ([#537](https://github.com/coupergateway/couper/pull/537))
  * Config validation, e.g. label-uniqueness checks ([#563](https://github.com/coupergateway/couper/pull/563))
  * [OIDC](https://docs.couper.io/configuration/block/oidc) not using referenced backends, if only specific backends (`configuration_backend`, `jwks_uri_backend`, `token_backend`, `userinfo_backend`) were configured ([#570](https://github.com/coupergateway/couper/pull/570))
  * [OIDC](https://docs.couper.io/configuration/block/oidc) configuration related go-routine leak after retrieving a new payload due to config ttl ([#564](https://github.com/coupergateway/couper/pull/564))

* **Removed**
  * Endpoint path normalization to better match OpenAPI behavior ([#526](https://github.com/coupergateway/couper/pull/526))

## [1.9.2](https://github.com/coupergateway/couper/releases/tag/v1.9.2)

* **Fixed**
  * configuration related panic while loading backends with [`oauth2` block](https://docs.couper.io/configuration/block/oauth2_req_auth) which depends on other defined backends ([#524](https://github.com/coupergateway/couper/pull/524))
  * erroneous retries for [`oauth2`](https://docs.couper.io/configuration/block/oauth2_req_auth) backend authorization ([#529](https://github.com/coupergateway/couper/pull/529))
    * with `retries = 0` ([#528](https://github.com/coupergateway/couper/pull/528))
    * with `retries` > `0` and related origin configuration ([#529](https://github.com/coupergateway/couper/pull/529))
  * race condition resulting in empty [`backends.<label>.health.state` variable](https://docs.couper.io/configuration/block/health) ([#530](https://github.com/coupergateway/couper/pull/530))
  * enabled json html escaping inherited from Go lib ([#531](https://github.com/coupergateway/couper/pull/531))

## [1.9.1](https://github.com/coupergateway/couper/releases/tag/v1.9.1)

* **Fixed**
  * Missing environment key error while using multiple configuration files ([#522](https://github.com/coupergateway/couper/pull/522))

## [1.9.0](https://github.com/coupergateway/couper/releases/tag/v1.9.0)

Couper 1.9 is a feature release bringing more comfort and enhanced stability to
the Couper configuration. It also improves the permission handling and provides a
couple of bug fixes. For a complete list of changes see below.

As of release 1.9 it is possible to split a Couper configuration into multiple
`.hcl`-files. You can now, for example, use different configuration files for
your `api`, `files` and `definitions` blocks, or keep your development, testing
and production setups separated. All the configuration files given at
[startup](https://docs.couper.io/configuration/command-line) will be [merged together](https://docs.couper.io/configuration/multiple-files).

The new block [`beta_health`](https://docs.couper.io/configuration/block/health) ([beta](https://docs.couper.io/getting-started/beta-features))
allows you to configure recurring health check requests for a backend.
By default, Couper won't request backends considered unhealthy which might help
them recover due to the reduced amount of requests.
The current health state of a backend can be accessed by variable.
Changes in healthiness will be [logged](https://docs.couper.io/observation/logging) and exported as [metrics](https://docs.couper.io/observation/metrics).

To make permission handling easier to grasp we've dropped the term `scope` and
accordingly changed the names of the `beta_scope`, `beta_scope_claim` and `beta_scope_map`
attributes to `beta_required_permission`, `beta_permissions_claim` and `beta_permissions_map`,
respectively. Furthermore, `beta_required_permission` (formerly `beta_scope`) can now
be an HCL expression. If `beta_required_permission` is specified in both an `endpoint`
and its parent `api` block, the former *overrides* the latter.
Our permission handling examples illustrate some common use cases:
[basic example](https://github.com/coupergateway/couper-examples/tree/master/permissions),
[roles example](https://github.com/coupergateway/couper-examples/tree/master/permissions-rbac),
[map example](https://github.com/coupergateway/couper-examples/tree/master/permissions-map)

Along with this release goes the latest [extension for VSCode](https://marketplace.visualstudio.com/items?itemName=couper.couperconf)
which now indicates misplaced blocks and attributes, missing block labels and so on.
We've also updated the completion suggestions and fixed a couple of syntax highlighting
issues.

* **Added**
  * Couper now [reads and merges multiple configuration files](https://docs.couper.io/configuration/multiple-files) ([#437](https://github.com/coupergateway/couper/pull/437), [#515](https://github.com/coupergateway/couper/pull/515))
  * `beta_health`-block to `backend`-block to enable continuous health-checks for defined backends ([#313](https://github.com/coupergateway/couper/pull/313))
    * `backends.<name>.health` variable to access the current health-check state _(subject to change)_
  * Log malformed duration settings ([#487](https://github.com/coupergateway/couper/pull/487))
  * `url` attribute could make use of our wildcard pattern `/**` and relative urls in combination with a backend reference ([#480](https://github.com/coupergateway/couper/pull/480))
  * `jwks_max_stale` in [`jwt` block](./docs/REFERENCE.md#jwt-block) ([#502](https://github.com/coupergateway/couper/pull/502))
  * `jwks_ttl`, `jwks_max_stale` and `configuration_max_stale` in [`oidc` block](./docs/REFERENCE.md#oidc-block) ([#502](https://github.com/coupergateway/couper/pull/502))
  * Error handling for `backend`, `backend_openapi_validation` and `backend_timeout` [error types](./docs/ERRORS.md) ([#490](https://github.com/coupergateway/couper/pull/490))
  * `response.bytes` log-field to backend logs if read from body, fallback is the `Content-Length` header ([#494](https://github.com/coupergateway/couper/pull/494))
  * [Error types](./docs/ERRORS.md) `endpoint` and `access_control` ([#500](https://github.com/coupergateway/couper/pull/500))

* **Changed**
  * Permission handling: ([#477](https://github.com/coupergateway/couper/pull/477), [#504](https://github.com/coupergateway/couper/pull/504))
    * renamed `beta_scope` attribute for [`api`](./docs/REFERENCE.md#api-block) and [`endpoint`](./docs/REFERENCE.md#endpoint-block) blocks to `beta_required_permission`; `beta_required_permission` in `endpoint` now *overriding* `beta_required_permission` in containing `api` block; allowing an expression as attribute value
    * renamed `beta_scope_claim` and `beta_scope_map` attributes for [`jwt` block](./docs/REFERENCE.md#jwt-block) to `beta_permissions_claim` and `beta_permissions_map`
    * removed `beta_operation_denied` and `beta_scope` [error types](./docs/ERRORS.md#api-and-endpoint-error-types)
    * renamed `beta_insufficient_scope` [error type](./docs/ERRORS.md#api-and-endpoint-error-types) to `beta_insufficient_permissions`
    * added `request.context.beta_required_permission` and `request.context.beta_granted_permissions` [request variables](./docs/REFERENCE.md#request)
  * Clarified the type of various [attributes/variables](./docs/REFERENCE.md) ([#485](https://github.com/coupergateway/couper/pull/485))
  * [`spa` block](./docs/REFERENCE.md#spa-block) can be defined multiple times now ([#510](https://github.com/coupergateway/couper/pull/510))
  * [`files` block](./docs/REFERENCE.md#files-block) can be defined multiple times now ([#513](https://github.com/coupergateway/couper/pull/513))

* **Fixed**
  * Keys in object type attribute values are only handled case-insensitively if reasonable (e.g. they represent HTTP methods or header field values) ([#461](https://github.com/coupergateway/couper/pull/461))
  * Multiple labels for [`error_handler` blocks](./docs/ERRORS.md#error_handler-specification) ([#462](https://github.com/coupergateway/couper/pull/462))
  * [`error_handler` blocks](./docs/REFERENCE.md#error-handler-block) for an error type defined in both `endpoint` and `api` ([#469](https://github.com/coupergateway/couper/pull/469))
  * Request methods are treated case-insensitively when comparing them to methods in the `allowed_methods` attribute of [`api`](./docs/REFERENCE.md#api-block) or [`endpoint`](./docs/REFERENCE.md#endpoint-block) blocks ([#478](https://github.com/coupergateway/couper/pull/478))
  * Do not allow multiple `backend` blocks in `proxy` and `request` blocks ([#483](https://github.com/coupergateway/couper/pull/483))
  * Panic if an [`error_handler` block](./docs/REFERENCE.md#error-handler-block) following another `error_handler` block has no label ([#486](https://github.com/coupergateway/couper/pull/486))
  * Spurious `duplicate endpoint /**` error for APIs sharing the same base path ([#507](https://github.com/coupergateway/couper/pull/507))
  * Invalid (by [OpenAPI validation](./docs/REFERENCE.md#openapi-block)) backend response missing in [`backend_responses`](./docs/REFERENCE.md#backend_responses) ([#501](https://github.com/coupergateway/couper/pull/501))
  * Ignore the `expected_status` check for a request configured via a [`proxy`](./docs/REFERENCE.md#proxy-block) or [`request`](./docs/REFERENCE.md#request-block) block if a [`backend` error](./docs/ERRORS.md#endpoint-error-types) occurred ([#505](https://github.com/coupergateway/couper/pull/505))
  * `merge()` function removes key with `null` value. ([#518](https://github.com/coupergateway/couper/pull/518))

* **Removed**
  * support for `beta_oidc` block (use [`oidc` block](./docs/REFERENCE.md#oidc-block) instead) ([#475](https://github.com/coupergateway/couper/pull/475))
  * support for `beta_oauth_authorization_url` and `beta_oauth_verifier` functions (use `oauth2_authorization_url` and `oauth2_verifier` [functions](./docs/REFERENCE.md#functions) instead) ([#475](https://github.com/coupergateway/couper/pull/475))
  * `path` attribute from `endpoint` (and `proxy`) block; use `path` attribute in `backend` block instead ([#516](https://github.com/coupergateway/couper/pull/516))

## [1.8.1](https://github.com/coupergateway/couper/releases/tag/v1.8.1)

* **Fixed**
  * missing error handling while loading a given `ca_file` ([#460](https://github.com/coupergateway/couper/pull/460))
  * allow [`api` blocks](./docs/REFERENCE.md#api-block) sharing the same `base_path` ([#471](https://github.com/coupergateway/couper/pull/471))

## [1.8.0](https://github.com/coupergateway/couper/releases/tag/v1.8.0)

* **Added**
  * `disable_private_caching` attribute for the [JWT Block](./docs/REFERENCE.md#jwt-block) ([#418](https://github.com/coupergateway/couper/pull/418))
  * [`backend_request`](./docs/REFERENCE.md#backend_request) and [`backend_response`](./docs/REFERENCE.md#backend_response) variables ([#430](https://github.com/coupergateway/couper/pull/430))
  * `beta_scope_map` attribute for the [JWT Block](./docs/REFERENCE.md#jwt-block) ([#434](https://github.com/coupergateway/couper/pull/434))
  * `saml` [error type](./docs/ERRORS.md#error-types) ([#424](https://github.com/coupergateway/couper/pull/424))
  * `allowed_methods` attribute for the [API](./docs/REFERENCE.md#api-block) or [Endpoint Block](./docs/REFERENCE.md#endpoint-block) ([#444](https://github.com/coupergateway/couper/pull/444))
  * new HCL [functions](./docs/REFERENCE.md#functions): `contains()`, `join()`, `keys()`, `length()`, `lookup()`, `set_intersection()`, `to_number()` ([#455](https://github.com/coupergateway/couper/pull/455))
  * `ca_file` option to `settings` (also as argument and environment option) ([#447](https://github.com/coupergateway/couper/pull/447))
    * Option for adding the given PEM encoded ca-certificate to the existing system certificate pool for all outgoing connections.

* **Changed**
  * Automatically add the `private` directive to the response `Cache-Control` HTTP header field value for all resources protected by [JWT](./docs/REFERENCE.md#jwt-block) ([#418](https://github.com/coupergateway/couper/pull/418))

* **Fixed**
  * improved protection against sniffing using unauthorized requests with non-standard method to non-existent endpoints in protected API ([#441](https://github.com/coupergateway/couper/pull/441))
  * Couper handles OS-Signal `INT` in all cases in combination with the `-watch` argument ([#456](https://github.com/coupergateway/couper/pull/456))
  * some [error types](./docs/ERRORS.md#access-control-error-types) related to [JWT](./docs/REFERENCE.md#jwt-block) ([#438](https://github.com/coupergateway/couper/pull/438))

## [1.7.2](https://github.com/coupergateway/couper/releases/tag/v1.7.2)

* **Fixed**
  * free up resources for backend response bodies without variable reference ([#449](https://github.com/coupergateway/couper/pull/449))
  * Linux and Windows binary version output (CI) ([#446](https://github.com/coupergateway/couper/pull/446))
  * error handling for empty `error_handler` labels ([#432](https://github.com/coupergateway/couper/pull/432))

## [1.7.1](https://github.com/coupergateway/couper/releases/tag/v1.7.1)

* **Fixed**
  * missing upstream log field value for [`request.proto`](./docs/LOGS.md#backend-fields) ([#421](https://github.com/coupergateway/couper/pull/421))
  * handling of `for` loops in HCL ([#426](https://github.com/coupergateway/couper/pull/426))
  * handling of conditionals in HCL: only predicates evaluating to boolean are allowed ([#429](https://github.com/coupergateway/couper/pull/429))
  * broken binary on macOS Monterey; build with latest go 1.17.6 (ci) ([#439](https://github.com/coupergateway/couper/pull/439))

## [1.7](https://github.com/coupergateway/couper/releases/tag/v1.7.0)

We start 2022 with fresh release of Couper with some exciting features.

Our **OpenID-Connect** (OIDC) configuration specification has been proven as final and is moved out of beta to the [`oidc` block](./docs/REFERENCE.md#oidc-block).
(Couper will still support `beta_oidc` until version `1.8`). With OIDC, Couper supports a variety of Identity Provides such as Google, Azure AD, Keycloak and many more.

While microservices aim for decoupling, they still need to work _together_. A typical API gateway approach is to make them individually accessible and move the point of integration into the client. Couper **sequences** however allows you to chain requests _in the gateway_. The response of one service call is used as input for the request to the next service. This keeps coupling loose and inter-service connectivity robust.
How Couper can help here is explained in our [sequence example](https://github.com/coupergateway/couper-examples/tree/master/sequences).

As part of our efforts to ease observability, Couper now allows you to collect **custom log data**. Use the [`custom_log_fields` attribute](./docs/LOGS.md#custom-logging)
all over your configuration file to augment your logs with information that is relevant to your application. Check out our [example](https://github.com/coupergateway/couper-examples/tree/master/custom-logging) to find out how it works.

To further improve the developer experience with Couper the [container image](https://hub.docker.com/r/coupergateway/couper) supports `amd64` and `arm64` architecture now.
On top of that the binary installation has been improved for [homebrew](https://brew.sh/) users: `brew tap coupergateway/couper && brew install couper` and go!

* **Added**
  * Support for [sequences](./docs/REFERENCE.md#endpoint-sequence) of outgoing endpoint requests ([#405](https://github.com/coupergateway/couper/issues/405))
  * `expected_status` attribute for `request` and `proxy` block definitions which can be caught with [error handling](./docs/ERRORS.md#endpoint-related-error_handler) ([#405](https://github.com/coupergateway/couper/issues/405))
  * [`custom_log_fields`](./docs/LOGS.md#custom-logging) attribute to be able to describe a user defined map for `custom` log field enrichment ([#388](https://github.com/coupergateway/couper/pull/388))
  * [`jwt` block](./docs/REFERENCE.md#jwt-block)/[`jwt_signing_profile` block](./docs/REFERENCE.md#jwt-signing-profile-block) support ECDSA signatures ([#401](https://github.com/coupergateway/couper/issues/401))
  * `user` as context variable from a [Basic Auth](./docs/REFERENCE.md#basic-auth-block) is now accessible via `request.context.<label>.user` for successfully authenticated requests ([#402](https://github.com/coupergateway/couper/pull/402))

* **Changed**
  * [`oidc` block](./docs/REFERENCE.md#oidc-block) is out of [beta](./docs/BETA.md). (The `beta_oidc` block name will be removed with Couper 1.8. ([#400](https://github.com/coupergateway/couper/pull/400))
  * `oauth2_authorization_url()` and `oauth2_verifier()` [functions](./docs/REFERENCE.md#functions) are our of beta. (The old function names `beta_oauth_...` will be removed with Couper 1.8). ([#400](https://github.com/coupergateway/couper/pull/400))
  * The access control for the OIDC redirect endpoint ([`oidc` block](./docs/REFERENCE.md#oidc-block)) now verifies ID token signatures ([#404](https://github.com/coupergateway/couper/pull/404))
  * `header = "Authorization"` is now the default token source for [JWT](./docs/REFERENCE.md#jwt-block) and may be omitted ([#413](https://github.com/coupergateway/couper/issues/413))
  * Improved the validation for unique keys in all map-attributes in the config ([#403](https://github.com/coupergateway/couper/pull/403))
  * Missing [scope or roles claims](./docs/REFERENCE.md#jwt-block), or scope or roles claim with unsupported values are now ignored instead of causing an error ([#380](https://github.com/coupergateway/couper/issues/380))
  * Unbeta [OIDC block](./docs/REFERENCE.md#oidc-block). The old block name is still usable with Couper 1.7, but will no longer work with Couper 1.8. ([#400](https://github.com/coupergateway/couper/pull/400))
  * Unbeta the `oauth2_authorization_url()` and `oauth2_verifier()` [function](./docs/REFERENCE.md#functions). The prefix is changed from `beta_oauth_...` to `oauth2_...`. The old function names are still usable with Couper 1.7, but will no longer work with Couper 1.8. ([#400](https://github.com/coupergateway/couper/pull/400))
  * Automatically add the `private` directive to the response `Cache-Control` HTTP header field value for all resources protected by [JWT](./docs/REFERENCE.md#jwt-block) ([#418](https://github.com/coupergateway/couper/pull/418))

* **Fixed**
  * build-date configuration for binary and docker builds ([#396](https://github.com/coupergateway/couper/pull/396))
  * exclude file descriptor limit startup-logs for Windows ([#396](https://github.com/coupergateway/couper/pull/396), [#383](https://github.com/coupergateway/couper/pull/383))
  * possible race conditions while updating JWKS for the [JWT access control](./docs/REFERENCE.md#jwt-block) ([#398](https://github.com/coupergateway/couper/pull/398))
  * panic while accessing primitive variables with a key ([#377](https://github.com/coupergateway/couper/issues/377))
  * [`default()`](./docs/REFERENCE.md#functions) function continues to the next fallback value if this is a string type and an argument evaluates to an empty string ([#408](https://github.com/coupergateway/couper/issues/408))
  * missing read of client-request bodies if related variables are used in referenced access controls only (e.g. JWT token source) ([#415](https://github.com/coupergateway/couper/pull/415))

* **Dependencies**
  * Update [kin-openapi](https://github.com/getkin/kin-openapi) used for [OpenAPI](./docs/REFERENCE.md#openapi-block) validation to `v0.83.0` ([#399](https://github.com/coupergateway/couper/pull/399))

## [1.6](https://github.com/coupergateway/couper/releases/tag/1.6)

* **Added**
  * Register `default` function as `coalesce` alias ([#356](https://github.com/coupergateway/couper/pull/356))
  * New HCL function [`relative_url()`](./docs/REFERENCE.md#functions) ([#361](https://github.com/coupergateway/couper/pull/361))
  * Log file descriptor limit at startup ([#383](https://github.com/coupergateway/couper/pull/383))
  * [`error_handler`](/docs/ERRORS.md) block support for `api` and `endpoint` blocks ([#317](https://github.com/coupergateway/couper/pull/317))
    * Enables reacting to additional [error types](/docs/ERRORS.md#error-types): `beta_scope`, `beta_insufficient_scope` and `beta_operation_denied`
  * `split()` and `substr()` [functions](./docs/REFERENCE.md#functions) ([#390](https://github.com/coupergateway/couper/pull/390))
  * hcl syntax verification for our configuration file ([#296](https://github.com/coupergateway/couper/pull/296)), ([#168](https://github.com/coupergateway/couper/issues/168)), ([#188](https://github.com/coupergateway/couper/issues/188))
    * validate against the schema and additional requirements
    * available as [`verify`](docs/CLI.md) command too

* **Changed**
  * [`server` block](./docs/REFERENCE.md#server-block) label is now optional, [`api` block](./docs/REFERENCE.md#api-block) may be labelled ([#358](https://github.com/coupergateway/couper/pull/358))
  * Timings in logs are now numeric values ([#367](https://github.com/coupergateway/couper/issues/367))

* **Fixed**
  * Handling of [`accept_forwarded_url`](./docs/REFERENCE.md#settings-block) "host" if `H-Forwarded-Host` request header field contains a port ([#360](https://github.com/coupergateway/couper/pull/360))
  * Setting `Vary` response header fields for [CORS](./docs/REFERENCE.md#cors-block) ([#362](https://github.com/coupergateway/couper/pull/362))
  * Use of referenced backends in [OAuth2 CC Blocks](./docs/REFERENCE.md#oauth2-cc-block) ([#321](https://github.com/coupergateway/couper/issues/321))
  * [CORS](./docs/REFERENCE.md#cors-block) preflight requests are not blocked by access controls anymore ([#366](https://github.com/coupergateway/couper/pull/366))
  * Reduced memory usage for backend response bodies which just get piped to the client and are not required to be read by Couper due to a variable references ([#375](https://github.com/coupergateway/couper/pull/375))
    * However, if a huge message body is passed and additionally referenced via e.g. `json_body`, Couper may require a lot of memory for storing the data structure.
  * For each SAML attribute listed in [`array_attributes`](./docs/REFERENCE.md#saml-block) at least an empty array is created in `request.context.<label>.attributes.<name>` ([#369](https://github.com/coupergateway/couper/pull/369))
  * HCL: Missing support for RelativeTraversalExpr, IndexExpr, UnaryOpExpr ([#389](https://github.com/coupergateway/couper/pull/389))
  * HCL: Missing support for different variable index key types ([#391](https://github.com/coupergateway/couper/pull/391))
  * [OIDC](./docs/REFERENCE.md#oidc-block): rejecting an ID token lacking an `aud` claim or with a `null` value `aud` ([#393](https://github.com/coupergateway/couper/pull/393))

## [1.5](https://github.com/coupergateway/couper/releases/tag/1.5)

* **Added**
  * `Accept: application/json` request header to the OAuth2 token request, in order to make the Github token endpoint respond with a JSON token response ([#307](https://github.com/coupergateway/couper/pull/307))
  * Documentation of [logs](docs/LOGS.md) ([#310](https://github.com/coupergateway/couper/pull/310))
  * `signing_ttl` and `signing_key`/`signing_key_file` to [`jwt` block](./docs/REFERENCE.md#jwt-block) for use with [`jwt_sign()` function](#functions) ([#309](https://github.com/coupergateway/couper/pull/309))
  * `jwks_url` and `jwks_ttl` to [`jwt` block](./docs/REFERENCE.md#jwt-block) ([#312](https://github.com/coupergateway/couper/pull/312))
  * `token_value` attribute in [`jwt` block](./docs/REFERENCE.md#jwt-block) ([#345](https://github.com/coupergateway/couper/issues/345))
  * `headers` attribute in [`jwt_signing_profile` block](./docs/REFERENCE.md#jwt-signing-profile-block) ([#329](https://github.com/coupergateway/couper/issues/329))

* **Changed**
  * Organized log format fields for uniform access and upstream log ([#300](https://github.com/coupergateway/couper/pull/300))
  * `claims` in a [`jwt` block](./docs/REFERENCE.md#jwt-block) are now evaluated per request, so that [`request` properties](./docs/REFERENCE.md#request) can be used as required claim values ([#314](https://github.com/coupergateway/couper/pull/314))
  * how Couper handles missing variables during context evaluation ([#255](https://github.com/coupergateway/couper/pull/225))
    * Previously missing elements results in evaluation errors and expressions like `set_response_headers` failed completely instead of one key/value pair.
      The evaluation has two steps now and will look up variables first and prepares the given expression to return `Nil` as fallback.

* **Fixed**
  * Key for storing and reading [OpenID configuration](./docs/REFERENCE.md#oidc-block) ([#319](https://github.com/coupergateway/couper/pull/319))

* [**Beta**](./docs/BETA.md)
  * `beta_scope_claim` attribute to [`jwt` block](./docs/REFERENCE.md#jwt-block); `beta_scope` attribute to [`api`](./docs/REFERENCE.md#api-block) and [`endpoint` block](./docs/REFERENCE.md#endpoint-block)s; [error types](./docs/ERRORS.md#error-types) `beta_operation_denied` and `beta_insufficient_scope` ([#315](https://github.com/coupergateway/couper/pull/315))
  * `beta_roles_claim` and `beta_roles_map` attributes to [`jwt` block](./docs/REFERENCE.md#jwt-block) ([#325](https://github.com/coupergateway/couper/pull/325)) ([#338](https://github.com/coupergateway/couper/pull/338)) ([#352](https://github.com/coupergateway/couper/pull/352))
  * Metrics: [Prometheus exporter](./docs/METRICS.md) ([#295](https://github.com/coupergateway/couper/pull/295))

* **Dependencies**
  * build with go 1.17 ([#331](https://github.com/coupergateway/couper/pull/331))

## [1.4](https://github.com/coupergateway/couper/releases/tag/1.4)

Release date: 2021-08-26

This release introduces [_Beta Features_](./docs/BETA.md). We use beta features to develop and experiment with new, complex features for you while still being able to maintain our compatibility promise. You can see beta features as a feature preview. To make users aware that a beta feature is used their configuration items are prefixed with `beta_`.

The first beta features incorporate the OAuth2 functionality into the Access Control capabilities of Couper. The [`beta_oauth2 {}` block](./docs/REFERENCE.md#oauth2-ac-block-beta) implements OAuth2 Authorization Code Grant Flows. The companion block [`beta_oidc {}`](./docs/REFERENCE.md#oidc-block) implements [OIDC](https://openid.net/connect/), which allows simple integration of 3rd-party systems such as Google, Github or Keycloak for SSO (Single-Sign-On).

Together with transparent [Websockets](docs/REFERENCE.md#websockets-block) support that you can enable in your `proxy {}` block, you can guard existing Web applications with Couper via OIDC.

To aid observability of your setups, Couper sends its request ID as the `Couper-Request-Id` HTTP header in both backend requests and client responses. This makes it possible to trace events and correlate logs throughout the service chain. Couper can also accept a request ID generated by a downstream system like for example a load balancer. Like all [settings](./docs/REFERENCE.md#settings-block), these can be configured in the config, as [command line flag](./docs/CLI.md) or via [environment variables](./DOCKER.md#environment-options).

Load balancers or ingress services often provide `X-Forwarded-Host` headers. Couper can be configured to use these to change the properties of the `request` variable. This allows a Couper configuration to adapt to the run time environment, for example to create a back link for OIDC or SAML authorization requests with the `request.origin` variable.

If your applications are running in multiple setups, like testing and production environments, there will likely be more parameters that you want to have configurable. Backend origins, user names, credentials, timeouts, all that could be nice to be changed without a new deployment. Couper supports using environment variables with `env.VAR`-like expressions. Now, Couper can also provide [default values](./docs/REFERENCE.md#defaults-block) for those variables. This makes it easy to have values configurable without the need to provide values outside of Couper (e.g. in Kubernetes). Our [env vars example](https://github.com/coupergateway/couper-examples/blob/master/env-var/) shows that in action.

* **Added**
  * `environment_variables` map in the [`defaults`](./docs/REFERENCE.md#defaults-block) block to define default values for environment variables ([#271](https://github.com/coupergateway/couper/pull/271))
  * [`https-dev-proxy` option](./docs/REFERENCE.md#settings-block) creates a TLS server listing on the given TLS port. Requests are forwarded to the given `server` port. The certificate is generated on-the-fly. This function is intended for local development setups to support browser features requiring HTTPS connections, such as secure cookies. ([#281](https://github.com/coupergateway/couper/pull/281))
  * [`websockets`](docs/REFERENCE.md#websockets-block) option in `proxy` block enables transparent websocket support when proxying to upstream backends ([#198](https://github.com/coupergateway/couper/issues/198))
  * Client request [variables](./docs/REFERENCE.md#request) `request.url`, `request.origin`, `request.protocol`, `request.host` and `request.port` ([#255](https://github.com/coupergateway/couper/pull/255))
  * [Command-line option](https://docs.couper.io/configuration/command-line#surrounding-architecture-options) `-accept-forwarded-url` and [setting](https://docs.couper.io/configuration/block/settings) `accept_forwarded_url` to accept `proto`, `host`, or `port` from `X-Forwarded-Proto`, `X-Forwarded-Host` or `X-Forwarded-Port` request headers ([#255](https://github.com/coupergateway/couper/pull/255))
  * Couper sends its request ID as `Couper-Request-Id` HTTP header in backend requests and client responses. This can be configured with the `request_id_backend_header` and `request_id_client_header` [settings](./docs/REFERENCE.md#settings-block) ([#268](https://github.com/coupergateway/couper/pull/268))
  * [`request_id_accept_from_header` setting](./docs/REFERENCE.md#settings-block) configures Couper to use a downstream request ID instead of generating its own in order to help correlating log events across services ([#268](https://github.com/coupergateway/couper/pull/268))
  * [`couper.version` variable](docs/REFERENCE.md#couper) ([#274](https://github.com/coupergateway/couper/pull/274))
  * `protocol`, `host`, `port`, `origin`, `body`, `json_body` to [`backend_requests` variable](./docs/REFERENCE.md#backend_requests) ([#278](https://github.com/coupergateway/couper/pull/278))
  * Locking to avoid concurrent requests to renew [OAuth2 Client Credentials](./docs/REFERENCE.md#oauth2-cc-block) access tokens ([#270](https://github.com/coupergateway/couper/issues/270))
  * `log-level` in the [`settings`](./docs/REFERENCE.md#settings-block) block to define when a log is printed ([#306](https://github.com/coupergateway/couper/pull/306))

* **Changed**
  * The `sp_acs_url` in the [SAML Block](./docs/REFERENCE.md#saml-block) may now be relative ([#265](https://github.com/coupergateway/couper/pull/265))

* **Fixed**
  * No GZIP compression for small response bodies ([#186](https://github.com/coupergateway/couper/issues/186))
  * Missing error type for [request](docs/REFERENCE.md#request-block)/[response](docs/REFERENCE.md#response-block) body, json_body or form_body related HCL evaluation errors ([#276](https://github.com/coupergateway/couper/pull/276))
  * [`request.url`](./docs/REFERENCE.md#request) and [`backend_requests.<label>.url`](./docs/REFERENCE.md#backend_requests) now contain a query string if present ([#278](https://github.com/coupergateway/couper/pull/278))
  * [`backend_responses.<label>.status`](./docs/REFERENCE.md#backend_responses) is now integer ([#278](https://github.com/coupergateway/couper/pull/278))
  * [`backend_requests.<label>.form_body`](./docs/REFERENCE.md#backend_requests) was always empty ([#278](https://github.com/coupergateway/couper/pull/278))
  * Documentation of [`request.query.<name>`](./docs/REFERENCE.md#request) ([#278](https://github.com/coupergateway/couper/pull/278))
  * Missing access log on some error cases ([#267](https://github.com/coupergateway/couper/issues/267))
  * Panic during backend origin / url usage with previous parse error ([#206](https://github.com/coupergateway/couper/issues/206))
  * [Basic Auth](./docs/REFERENCE.md#basic-auth-block) did not work if only the `htpasswd_file` attribute was defined ([#293](https://github.com/coupergateway/couper/pull/293))
  * Missing error handling for backend gzip header reads ([#291](https://github.com/coupergateway/couper/pull/291))
  * ResponseWriter fallback for possible statusCode 0 writes ([#291](https://github.com/coupergateway/couper/pull/291))
  * ResponseWriter buffer behaviour; prepared chunk writes ([#301](https://github.com/coupergateway/couper/pull/301))
  * Proper client-request canceling ([#294](https://github.com/coupergateway/couper/pull/294))

* [**Beta**](./docs/BETA.md)
  * OAuth2 Authorization Code Grant Flow: [`beta_oauth2 {}` block](./docs/REFERENCE.md#oauth2-ac-block-beta);  [`beta_oauth_authorization_url()`](./docs/REFERENCE.md#functions) and [`beta_oauth_verifier()`](./docs/REFERENCE.md#functions) ([#247](https://github.com/coupergateway/couper/pull/247))
  * OIDC Authorization Code Grant Flow: [`beta_oidc {}` block](./docs/REFERENCE.md#oidc-block) ([#273](https://github.com/coupergateway/couper/pull/273))

## [1.3.1](https://github.com/coupergateway/couper/compare/1.3...1.3.1)

* **Changed**
  * `Error` log-level for upstream responses with status `500` to `Info` log-level ([#258](https://github.com/coupergateway/couper/pull/258))

* **Fixed**
  * Missing support for `set_response_status` within a plain `error_handler` block ([#257](https://github.com/coupergateway/couper/pull/257))
  * Panic in jwt_sign() and saml_sso_url() functions without proper configuration ([#243](https://github.com/coupergateway/couper/issues/243))

## [1.3](https://github.com/coupergateway/couper/compare/1.2...1.3)

* **Added**
  * Modifier (`set/add/remove_form_params`) for the form parameters ([#223](https://github.com/coupergateway/couper/pull/223))
  * Modifier (`set_response_status`) to be able to modify the response HTTP status code ([#250](https://github.com/coupergateway/couper/pull/250))

* **Changed**
  * Stronger configuration check for `path` and `path_prefix` attributes, possibly resulting in configuration errors ([#232](https://github.com/coupergateway/couper/pull/232))
  * Modifier (`set/add/remove_response_headers`) is available for `api`, `files`, `server` and `spa` block, too ([#248](https://github.com/coupergateway/couper/pull/248))

* **Fixed**
  * The `path` field in the backend log ([#232](https://github.com/coupergateway/couper/pull/232))
  * Upstream requests with a known body-size have a `Content-Length` HTTP header field instead of `Transfer-Encoding: chunked` ([#163](https://github.com/coupergateway/couper/issues/163))
  * Exit endpoint if an error is occurred in `request` or `proxy` instead of processing a defined `response` ([#233](https://github.com/coupergateway/couper/pull/233))

## [1.2](https://github.com/coupergateway/couper/compare/1.1.1...1.2)

Release date: 2021-05-19

The most important feature of Couper 1.2 is the introduction of _custom
error handling_ in form of the [`error_handler`](/docs/ERRORS.md) block.
You can now register error handlers for [error types](/docs/ERRORS.md#error-types). Instead of the standard `error_file` template,
you can flexibly respond with arbitrary `response`s. `error_handler` is allowed in access control blocks (`jwt`, `saml2` …), where you
could e.g. handle missing tokens with a redirect-to-login. In the
future, `error_handler` will be usable in more config areas. Refer to
the [example](https://github.com/coupergateway/couper-examples/tree/master/error-handling-ba)
if you want to see it in action.

* **Added**
  * `error_handler` block for access controls ([#140](https://github.com/coupergateway/couper/pull/140))
  * `backend_responses.*.body` variable for accessing raw response body content ([#182](https://github.com/coupergateway/couper/issues/182))
  * more `oauth2` config options: `scope` and `token_endpoint_auth_method` (`client_secret_basic` or `client_secret_post`) ([#219](https://github.com/coupergateway/couper/pull/219), [#220](https://github.com/coupergateway/couper/pull/220))

* **Changed**
  * `saml2` fallback to `nameid-format:unspecified` ([#217](https://github.com/coupergateway/couper/pull/217))
  * `basic_auth` always responds with status code `401` ([#227](https://github.com/coupergateway/couper/pull/227))
  * `openapi` resolves relative server URLs to the current backend origin ([#230](https://github.com/coupergateway/couper/pull/230))

* **Fixed**
  * Fix `/healthz` route when called with `accept-encoding: gzip` ([#222](https://github.com/coupergateway/couper/pull/222))
  * Don't panic over duplicate access control definitions, log error instead ([#221](https://github.com/coupergateway/couper/pull/221))
  * Response for missing routes should have status code `404` ([#224](https://github.com/coupergateway/couper/pull/224))
  * Fix possible race-condition with concurrent `openapi` validations ([#231](https://github.com/coupergateway/couper/pull/231))
  * Fix use of server URLs without port in `openapi` ([#230](https://github.com/coupergateway/couper/pull/230))

## [1.1.1](https://github.com/coupergateway/couper/compare/1.1...1.1.1)

Release date: 2021-04-21

* **Fixed**
  * Endpoint responses are written and logged with correct status-code ([#216](https://github.com/coupergateway/couper/issues/216))
    * affected: a plain `response` without any additional headers or body configuration

## [1.1](https://github.com/coupergateway/couper/compare/1.0...1.1)

Release date: 2021-04-16

* **Fixed**
  * allow more +json mime types ([#207](https://github.com/coupergateway/couper/pull/207))
    * determines if ja request/response body gets parsed and provided as `json_body` variable
  * missing check for empty endpoint path patterns ([#211](https://github.com/coupergateway/couper/pull/211))
  * protected API (base)paths returns status 401 instead of 404 if a protected route was not found ([#211](https://github.com/coupergateway/couper/pull/211))
  * jwt source config definition ([#210](https://github.com/coupergateway/couper/issues/210))
  * missing inner context on context copy
  * possible panic for unhandled error template write errors ([#205](https://github.com/coupergateway/couper/pull/205))
  * backend reference usage with string label ([#189](https://github.com/coupergateway/couper/issues/189))
  * cli argument filtering ([#204](https://github.com/coupergateway/couper/issues/204))
  * misleading jwt rsa key error ([#203](https://github.com/coupergateway/couper/issues/203))
  * watch handling on stat errors ([#202](https://github.com/coupergateway/couper/pull/202))

* **Changed**
  * Change access control validation logging ([#199](https://github.com/coupergateway/couper/issues/199))
    * log the first occurred error instead of an array

* **Added**
  * Add OAuth2 token request retry option  ([#167](https://github.com/coupergateway/couper/issues/167)) ([#209](https://github.com/coupergateway/couper/issues/209))

## [1.0](https://github.com/coupergateway/couper/compare/0.9...1.0)

Release date: 2021-04-09

* **Added**
  * `couper help` and usage documentation ([#187](https://github.com/coupergateway/couper/issues/187))

* **Changed**
  * Ensure unique keys for set_* and add_* attributes ([#183](https://github.com/coupergateway/couper/pull/183))
  * split docker entrypoint and command ([#192](https://github.com/coupergateway/couper/issues/192))

* **Fixed**
  * Fix missing `backend.origin` attribute url validation ([#191](https://github.com/coupergateway/couper/issues/191))

## [0.9](https://github.com/coupergateway/couper/compare/0.8...0.9)

Release date: 2021-04-08

* **Fixed**
  * Log option for `json` formatted logs: ([#176](https://github.com/coupergateway/couper/issues/176))
    * configured parent key applies to (almost) all log fields

* **Changed**
  * Change variable names to more user-friendly ones ([#180](https://github.com/coupergateway/couper/issues/180)):
    * `req` -> `request`
    * `ctx` -> `context`
    * `bereq` -> *removed*
    * `beresp` -> *removed*
    * `bereqs` -> `backend_requests`
    * `beresps` -> `backend_responses`
  * Log option for parent fields are 'global' now ([#176](https://github.com/coupergateway/couper/issues/176))
    * `COUPER_ACCESS_LOG_PARENT_FIELD`, `COUPER_BACKEND_LOG_PARENT_FIELD` -> `COUPER_LOG_PARENT_FIELD`

* **Added**
  * watch option for given Couper configuration file ([#24](https://github.com/coupergateway/couper/issues/24))
    * use `-watch` or via environment `COUPER_WATCH=true` to watch for file changes
  * log option pretty print for `json` log-format ([#179](https://github.com/coupergateway/couper/pull/179))
    * `-log-pretty` to enable formatted and key colored logs

## [0.8](https://github.com/coupergateway/couper/compare/0.7.0...v0.8)

Release date: 2021-04-06

* **Fixed**
  * Some possible race conditions in combination with multiple `proxy` and/or `request`
  definitions are fixed ([#157](https://github.com/coupergateway/couper/issues/177)) ([#160](https://github.com/coupergateway/couper/issues/160))
  * Log endpoint related recovered panics
  * CORS behaviour: result is now only dependent on the config, not the actual request; fixed Vary headers ([#173](https://github.com/coupergateway/couper/issues/173))
  * Fix json type assumption ([#177](https://github.com/coupergateway/couper/issues/177))
    * `req.json_body` result is an empty object for specific types ([#165](https://github.com/coupergateway/couper/issues/165))
    * Empty json array encodes to `null`. ([#162](https://github.com/coupergateway/couper/issues/162))
  * Fix missing string conversion for evaluated number values ([#175](https://github.com/coupergateway/couper/issues/175))
  * Loading optional labels of same type
  * multiplexer behaviour with multiple servers and hosts ([#161](https://github.com/coupergateway/couper/issues/161))
  * Fix missing access_control for file handler ([#169](https://github.com/coupergateway/couper/issues/169))
  * 404 behaviour for access controlled endpoints:
  deny instead of 404 if the request matches the related base_path ([#143](https://github.com/coupergateway/couper/issues/143))

* **Changed**
  * Rename log type for backend requests: `couper_upstream` -> `couper_backend` ([#159](https://github.com/coupergateway/couper/pull/159)) ([#172](https://github.com/coupergateway/couper/pull/172))
  * Rename `post` variable to `form_body` ([#158](https://github.com/coupergateway/couper/pull/158))

* **Added**
  * Add `json_body` attribute for `request` and `response` block ([#158](https://github.com/coupergateway/couper/issues/158))
  * `bytes` log field to represent the body size

## [0.7.0](https://github.com/coupergateway/couper/compare/0.6.1...0.7.0)

Release date: 2021-03-23

* **Fixed**
  * Recover from possible request/proxy related panics ([#157](https://github.com/coupergateway/couper/pull/157)) ([#145](https://github.com/coupergateway/couper/pull/145))
  * Configuration related hcl merge with an empty attributes and nested blocks

* **Changed**
  * `backend` block attributes `basic_auth`, `path_prefix` and `proxy` hcl evaluation during runtime
  * `request` attributes hcl evaluation during runtime ([#152](https://github.com/coupergateway/couper/pull/152))
  * Change configuration in combination with URL and backend.origin ([#144](https://github.com/coupergateway/couper/issues/144))
    * `request` and `proxy` block can use the `url` attribute instead of define or reference a `backend`
    * same applies to `oauth2.token_endpoint`
  * no `X-Forwarded-For` header enrichment from couper `proxy` ([#139](https://github.com/coupergateway/couper/pull/139))
  * more log context for access control related errors ([#154](https://github.com/coupergateway/couper/issues/154))

* **Added**
  * `saml` 2.0 `access_control` support ([#113](https://github.com/coupergateway/couper/issues/113))
  * Add new `strip-secure-cookies` setting ([#147](https://github.com/coupergateway/couper/issues/147))
    * removes `Secure` flag from all `Set-Cookie` header
  * CORS support (`server`, `files`, `spa`) ([#134](https://github.com/coupergateway/couper/issues/134))
    * previously `api` only
  * `error_file` attribute for `endpoint` block
  * hcl functions:
    * `merge`
    * `url_encode`
  * `backend`
    * OAuth2 support ([#130](https://github.com/coupergateway/couper/issues/130))
      * grant_type: `client_credentials`
      * `token` memory storage with ttl
    * `path_prefix` attribute ([#138](https://github.com/coupergateway/couper/issues/138))

## [0.6.1](https://github.com/coupergateway/couper/compare/0.6...0.6.1)

Release date: 2021-03-15

* **Fixed**
  * Fix missing panic recovering for backend roundtrips ([#142](https://github.com/coupergateway/couper/issues/142))
    * Fix backend `timeout` behaviour
    * Add a more specific error message for proxy body copy errors

* **Changed**
  * Couper just passes the `X-Forwarded-For` header if any instead of adding the client remote addr to the list ([#139](https://github.com/coupergateway/couper/pull/139))

* **Added**
  * `url_encode` function for RFC 3986 string encoding ([#136](https://github.com/coupergateway/couper/pull/136))

## [0.6](https://github.com/coupergateway/couper/compare/0.5.1...0.6)

Release date: 2021-03-11

* **Breaking Change**
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

* **Changed**
  * Client-Request and upstream response body buffering by default
  * Server shutdown delay and deadline defaults to `0s` now and can be configured via [env](DOCKER.md) if required
  * Websocket connection upgrades in combination with `proxy {}` are disabled
    * we will add a proxy option for ws usage later on

* **Fixed**
  * An absolute path resolving for `*_file` configuration attributes ([#120](https://github.com/coupergateway/couper/pull/120))

* **Added**
  * Endpoint:
    * Add `proxy` block to reverse proxy the client request to the configured `backend`.
    * Add `request` block to send a simple upstream request. [Docs](docs/README.md#request-block)
    * Add `response` block to create a custom client response. [Docs](docs/README.md#response-block)
  * Add `jwt_sign()` function to be able to create and sign a token with a `jwt_signing_profile`. [Docs](docs/README.md#functions) ([#112](https://github.com/coupergateway/couper/issues/112))
  * Add `unixtime()` function for the current unix time in seconds ([#124](https://github.com/coupergateway/couper/issues/124))

* **Code Refactoring**
  * underlying code structure to represent an `endpoint` block with `proxy`, `request` and `response` configuration
  * hcl evaluation context as own 'container' with `context.Context` interface
  * test cleanups

* **Dependencies**
  * build with go 1.16
  * logrus to v1.8.1
  * hcl to v2.9.1
  * kin-openapi to v.0.49.0

## [0.5.1](https://github.com/coupergateway/couper/compare/0.5...0.5.1)

Release date: 2021-02-16

* **Added**
  * backend:
    * a user-friendly `basic_auth` option
    * backend `proxy` url, `disable_connection_reuse` and `http2` settings ([#108](https://github.com/coupergateway/couper/pull/108))
  * version command

* **Changed**
  * KeepAlive `60s` ([#108](https://github.com/coupergateway/couper/pull/108)), previously `15s`
  * Reject requests which hits an endpoint with basic-auth access-control, and the configured password evaluates to an empty string ([#115](https://github.com/coupergateway/couper/pull/115))

## [0.5](https://github.com/coupergateway/couper/compare/0.4.2...0.5)

Release date: 2021-01-29

* **Fixed**
  * Fix missing http.Hijacker interface to be able to handle websocket upgrades ([#80](https://github.com/coupergateway/couper/issues/80))

* **Added**
  * Add additional eval functions: coalesce, json_decode, json_encode ([#105](https://github.com/coupergateway/couper/pull/105))
  * Add multi API support ([#103](https://github.com/coupergateway/couper/issues/103))
  * Add free endpoints ([#90](https://github.com/coupergateway/couper/issues/90))
  * Add remove_, set_ and  add_headers ([#98](https://github.com/coupergateway/couper/issues/98))

* **Code Refactoring**
  * improved internals for configuration load

* **Dependencies**
  * Upgrade hcl to 2.8.2
  * Upgrade go-cty module to 1.5.0
  * Upgrade logrus module to 1.7.0
  * Upgrade kin-openapi module to v0.37

## [0.4.2](https://github.com/coupergateway/couper/compare/0.4.1...0.4.2)

Release date: 2021-01-19

* **Fixed**
  * Fix used backend hash not dependent on (hcl) config hierarchy (transport key)
  * Fix logging http scheme even without a successful tls handshake ([#99](https://github.com/coupergateway/couper/pull/99))
  * Fix hcl.Body content for reference backends ([#96](https://github.com/coupergateway/couper/issues/96))

## [0.4.1](https://github.com/coupergateway/couper/compare/0.4...0.4.1)

Release date: 2021-01-18

* **Fixed**
  * Fix path trailing slash ([#94](https://github.com/coupergateway/couper/issues/94))
  * Fix query encoding ([#93](https://github.com/coupergateway/couper/issues/93))
  * Fix log_format (settings) configuration ([#61](https://github.com/coupergateway/couper/issues/61))

## [0.4](https://github.com/coupergateway/couper/compare/v0.3...0.4)

Release date: 2021-01-13

* **Added**
  * url log field ([#87](https://github.com/coupergateway/couper/issues/87))
  * Add proxy from env settings option ([#84](https://github.com/coupergateway/couper/issues/84))
  * Add backend settings:  `disable_certificate_validation`, `max_connections` ([#86](https://github.com/coupergateway/couper/issues/86))

* **Fixed**
  * command flag filter for bool values ([#85](https://github.com/coupergateway/couper/issues/85))
  * different proxy options for same origin should be part of the origin transport key

* **Code Refactoring**
  * configuration load and prepare related body merges on hcl level

## [0.3](https://github.com/coupergateway/couper/compare/v0.2...v0.3)

Release date: 2020-12-15

* **Added**
  * build version to startup log
  * upstream request/response validation with `openapi` ([#21](https://github.com/coupergateway/couper/issues/21)) ([#22](https://github.com/coupergateway/couper/issues/22))
  * request-id: uuid v4 format option [#31](https://github.com/coupergateway/couper/issues/31) ([#53](https://github.com/coupergateway/couper/issues/53))
  * `path_params` [#59](https://github.com/coupergateway/couper/issues/59)
  * gzip support ([#66](https://github.com/coupergateway/couper/issues/66))
  * `query_params` ([#73](https://github.com/coupergateway/couper/issues/73))
  * `json_body` access for request and response bodies [#44](https://github.com/coupergateway/couper/issues/44) ([#60](https://github.com/coupergateway/couper/issues/60))

* **Changed**
  * start Couper via `run` command now
  * internal router [#59](https://github.com/coupergateway/couper/issues/59)
  * docker tag behaviour on release [#70](https://github.com/coupergateway/couper/issues/70) ([#82](https://github.com/coupergateway/couper/issues/82))
  * request/response_headers to use `set` prefix ([#77](https://github.com/coupergateway/couper/issues/77))
  * passing the filename to underlying hcl diagnostics
  * Dockerfile to provide simple file serving ([#63](https://github.com/coupergateway/couper/issues/63))

* **Fixed**
  * handling cty null or unknown values during roundtrip eval [#71](https://github.com/coupergateway/couper/issues/71)
  * logging: start-time measurement
  * missing `backend.hostname` documentation ([#62](https://github.com/coupergateway/couper/issues/62))

## [0.2](https://github.com/coupergateway/couper/compare/v0.1...v0.2)

Release date: 2020-10-08

* **Added**
  * health check ([#29](https://github.com/coupergateway/couper/issues/29))
  * Basic-Auth support ([#19](https://github.com/coupergateway/couper/issues/19))
  * post (form) parsing for use in config variables ([#26](https://github.com/coupergateway/couper/issues/26))
  * more documentation

* **Fixed**
  * wildcard path join with trailing slash and respect req path ([#45](https://github.com/coupergateway/couper/pull/45))
  * env var mapping ([#35](https://github.com/coupergateway/couper/pull/35))
  * JWT HMAC keys ([#32](https://github.com/coupergateway/couper/pull/32))

## 0.1

Release date: 2020-09-11

* **Added**
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
