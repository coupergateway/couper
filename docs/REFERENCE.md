# Reference

- [Reference](#reference)
  - [Block Reference](#block-reference)
    - [Server Block](#server-block)
    - [Files Block](#files-block)
    - [SPA Block](#spa-block)
    - [API Block](#api-block)
    - [Endpoint Block](#endpoint-block)
    - [Proxy Block](#proxy-block)
    - [Request Block](#request-block)
    - [Response Block](#response-block)
    - [Backend Block](#backend-block)
      - [Duration](#duration)
    - [OpenAPI Block](#openapi-block)
    - [CORS Block](#cors-block)
    - [OAuth2 CC Block](#oauth2-cc-block)
    - [Definitions Block](#definitions-block)
    - [Basic Auth Block](#basic-auth-block)
    - [JWT Block](#jwt-block)
    - [JWT Signing Profile Block](#jwt-signing-profile-block)
    - [OAuth2 AC Block (Beta)](#oauth2-ac-block-beta)
    - [OIDC Block (Beta)](#oidc-block-beta)
    - [SAML Block](#saml-block)
    - [Settings Block](#settings-block)
    - [Defaults Block](#defaults-block)
  - [Access Control](#access-control)
  - [Health-Check](#health-check)
  - [Variables](#variables)
    - [couper](#couper)
    - [env](#env)
    - [request](#request)
    - [backend_requests](#backend_requests)
    - [backend_responses](#backend_responses)
  - [Functions](#functions)
  - [Modifiers](#modifiers)
    - [Request Header](#request-header)
    - [Response Header](#response-header)
    - [Set Response Status](#set-response-status)
  - [Parameters](#parameters)
    - [Query Parameter](#query-parameter)
    - [Form Parameter](#form-parameter)
    - [Path Parameter](#path-parameter)

## Block Reference

### Server Block

The `server` block is one of the root configuration blocks of Couper's configuration file.

| Block name | Context | Label            | Nested block(s) |
| :--------- | :------ | :--------------- | :-------------- |
| `server`   | -       | &#9888; required | [CORS Block](#cors-block), [Files Block](#files-block), [SPA Block](#spa-block) , [API Block(s)](#api-block), [Endpoint Block(s)](#endpoint-block) |

| Attribute(s)     | Type   | Default      | Description | Characteristic(s) | Example |
| :--------------- | :----- | :----------- | :---------- | :---------------- | :------ |
| `base_path`      | string | -            | Configures the path prefix for all requests. | &#9888; Inherited by nested blocks. | `base_path = "/api"` |
| `hosts`          | list   | port `:8080` | - | &#9888; required, if there is more than one `server` block. &#9888; Only one `hosts` attribute per `server` block is allowed. | `hosts = ["example.com", "localhost:9090"]` |
| `error_file`     | string | -            | Location of the error file template. | - | `error_file = "./my_error_page.html"` |
| `access_control` | list   | -            | Sets predefined [Access Control](#access-control) for `server` block context. | &#9888; Inherited by nested blocks. | `access_control = ["foo"]` |

### Files Block

The `files` block configures the file serving.

| Block name | Context                       | Label    | Nested block(s)           |
| :--------- | :---------------------------- | :------- | :------------------------ |
| `files`    | [Server Block](#server-block) | no label | [CORS Block](#cors-block) |

| Attribute(s)     | Type   | Default | Description | Characteristic(s) | Example |
| :--------------- | :----- | :------ | :---------- | :---------------- | :------ |
| `base_path`      | string | -       | Configures the path prefix for all requests. | - | `base_path = "/files"` |
| `document_root`  | string | -       | Location of the document root. | &#9888; required | `document_root = "./htdocs"` |
| `error_file`     | string | -       | Location of the error file template. | - | - |
| `access_control` | list   | -       | Sets predefined [Access Control](#access-control) for `files` block context. | - | `access_control = ["foo"]` |

### SPA Block

The `spa` block configures the Web serving for SPA assets.

| Block name | Context                       | Label    | Nested block(s)           |
| :--------- | :---------------------------- | :------- | :------------------------ |
| `spa`      | [Server Block](#server-block) | no label | [CORS Block](#cors-block) |

| Attribute(s)     | Type   | Default | Description | Characteristic(s) | Example |
| :--------------- | :----- | :------ | :---------- | :---------------- | :------ |
| `base_path`      | string | -       | Configures the path prefix for all requests. | - | `base_path = "/assets"` |
| `bootstrap_file` | string | -       | Location of the bootstrap file. | &#9888; required | `bootstrap_file = "./htdocs/index.html"` |
| `paths`          | list   | -       | List of SPA paths that need the bootstrap file. | &#9888; required | `paths = ["/app/**"]` |
| `access_control` | list   | -       | Sets predefined [Access Control](#access-control) for `spa` block context. | - | `access_control = ["foo"]` |

### API Block

The `api` block bundles endpoints under a certain `base_path`.

&#9888; If an error occurred for api endpoints the response gets processed
as json error with an error body payload. This can be customized via `error_file`.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`api`|[Server Block](#server-block)|Optional| [Endpoint Block(s)](#endpoint-block), [CORS Block](#cors-block)|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------  | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
|`base_path`|string|-|Configures the path prefix for all requests.|&#9888; Must be unique if multiple `api` blocks are defined.| `base_path = "/v1"`|
| `error_file` |string|-|Location of the error file template.|-|`error_file = "./my_error_body.json"`|
| `access_control` |list|-|Sets predefined [Access Control](#access-control) for `api` block context.|&#9888; Inherited by nested blocks.| `access_control = ["foo"]`|
| `beta_scope` |string or object|-|Scope value required to use this API (see [error type](../ERRORS.md#error-types) `beta_insufficient_scope`).|If the value is a string, the same scope value applies to all request methods. If there are different scope values for different request methods, use an object with the request methods as keys and string values. Methods not specified in this object are not permitted (see [error type](../ERRORS.md#error-types) `beta_operation_denied`). `"*"` is the key for "all other methods". A value `""` means "no (additional) scope required".| `beta_scope = "read"` or `beta_scope = { post = "write", "*" = "" }`|

### Endpoint Block

`endpoint` blocks define the entry points of Couper. The required _label_
defines the path suffix for the incoming client request. The `path` attribute
changes the path for the outgoing request (compare
[path mapping example](./README.md#routing-path-mapping)). Each `endpoint` block must
produce an explicit or implicit client response.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`endpoint`| [Server Block](#server-block), [API Block](#api-block) |&#9888; required, defines the path suffix for incoming client requests | [Proxy Block(s)](#proxy-block),  [Request Block(s)](#request-block), [Response Block](#response-block) |

<!-- TODO: decide how to place "modifier" in the reference table - same for other block which allow modifiers -->

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
|`request_body_limit`  |string|`64MiB`|Configures the maximum buffer size while accessing `request.form_body` or `request.json_body` content.|&#9888; Valid units are: `KiB, MiB, GiB`|`request_body_limit = "200KiB"`|
| `path`|string|-|Changeable part of the upstream URL. Changes the path suffix of the outgoing request.|-|-|
|`access_control`   |list|-|Sets predefined [Access Control](#access-control) for `endpoint` block context.|-| `access_control = ["foo"]`|
| `beta_scope` |string or object|-|Scope value required to use this endpoint (see [error type](../ERRORS.md#error-types) `beta_insufficient_scope`).|If the value is a string, the same scope value applies to all request methods. If there are different scope values for different request methods, use an object with the request methods as keys and string values. Methods not specified in this object are not permitted (see [error type](../ERRORS.md#error-types) `beta_operation_denied`). `"*"` is the key for "all other methods". A value `""` means "no (additional) scope required".| `beta_scope = "read"` or `beta_scope = { post = "write", "*" = "" }`|
|[Modifiers](#modifiers) |-|-|-|-|-|

### Proxy Block

The `proxy` block creates and executes a proxy request to a backend service.

&#9888; Multiple  `proxy` and [Request Block](#request-block)s are executed in parallel.
<!-- TODO: shorten label text in table below and find better explanation for backend, backend reference or url - same for request block-->

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`proxy`|[Endpoint Block](#endpoint-block)|&#9888; A `proxy` block or [Request Block](#request-block) w/o a label has an implicit label `"default"`. Only **one** `proxy` block or [Request Block](#request-block) w/ label `"default"` per [Endpoint Block](#endpoint-block) is allowed.|[Backend Block](#backend-block) (&#9888; required, if no [Backend Block](#backend-block) reference is defined or no `url` attribute is set.), [Websockets Block](#websockets-block) (&#9888; Either websockets attribute or block is allowed.)|

| Attribute(s) | Type | Default | Description | Characteristic(s) | Example |
| :----------- | :--- | :------ | :---------- | :---------------- | :------ |
| `websockets` | bool | false | Allows support for websockets. This attribute is only allowed in the 'default' `proxy` block. Other `proxy` blocks, [Request Blocks](#request-block) or [Response Blocks](#response-block) are not allowed in the current [Endpoint Block](#endpoint-block). | &#9888; Either websockets attribute or block is allowed. | `websockets = true` |
| `backend` |string|-|[Backend Block](#backend-block) reference, defined in [Definitions Block](#definitions-block)|&#9888; required, if no [Backend Block](#backend-block) or `url` attribute is defined.|`backend = "foo"`|
| `url` |string|-|If defined, the host part of the URL must be the same as the `origin` attribute of the [Backend Block](#backend-block) (if defined).|-|-|
|[Modifiers](#modifiers)|-|-|-|-|-|

### Request Block

The `request` block creates and executes a request to a backend service.

&#9888; Multiple [Proxy](#proxy-block) and `request` blocks are executed in parallel.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`request`| [Endpoint Block](#endpoint-block)|&#9888; A [Proxy Block](#proxy-block) or [Request Block](#request-block) w/o a label has an implicit label `"default"`. Only **one** [Proxy Block](#proxy-block) or [Request Block](#request-block) w/ label `"default"` per [Endpoint Block](#endpoint-block) is allowed.|[Backend Block](#backend-block) (&#9888; required, if no `backend` block reference is defined or no `url` attribute is set.|
<!-- TODO: add available http methods -->
| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `backend` |string|-|`backend` block reference, defined in [Definitions Block](#definitions-block)|&#9888; required, if no [Backend Block](#backend-block) is defined.|`backend = "foo"`|
| `url` |string|-|-|If defined, the host part of the URL must be the same as the `origin` attribute of the used [Backend Block](#backend-block) or [Backend Block Reference](#backend-block) (if defined).|-|
|`body`|string|-|-| Creates implicit default `Content-Type: text/plain` header field.|-|
|`json_body`|null, bool, number, string, map, list|-|-|Creates implicit default `Content-Type: text/plain` header field.|-|
| `form_body` |map|-|-|Creates implicit default `Content-Type: application/x-www-form-urlencoded` header field.|-|
|`method`    |string|`GET`|-|-|-|
|`headers`  |-|-|-|Same as `set_request_headers` in [Request Header](#request-header).|-|
|`query_params`|-|-|-|Same as `set_query_params` in [Query Parameter](#query-parameter).|-|

### Response Block

The `response` block creates and sends a client response.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`response`|[Endpoint Block](#endpoint-block)|no label|-|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `body`         | string|-|-|Creates implicit default `Content-Type: text/plain` header field.|-|
| `json_body`    | null, bool, number, string, map, list|-|-|Creates implicit default `Content-Type: application/json` header field.|-|
| `status`       | integer|`200`|HTTP status code.|-|-|
| `headers`      |string|-|Same as `set_response_headers` in [Request Header](#response-header).                  |-|-|

### Backend Block

The `backend` block defines the connection to a local/remote backend service.

&#9888; Backends can be defined in the [Definitions Block](#definitions-block) and referenced by _label_.

|Block name|Context|Label|Nested block(s)|
| :----------| :-----------| :-----------| :-----------|
|`backend`| [Definitions Block](#definitions-block), [Proxy Block](#proxy-block), [Request Block](#request-block)| &#9888; required, when defined in [Definitions Block](#definitions-block)| [OpenAPI Block](#openapi-block), [OAuth2 CC Block](#oauth2-cc-block)|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `basic_auth`                    | string|-|Basic auth for the upstream request. | format is `username:password`|-|
| `hostname`                      | string |-|Value of the HTTP host header field for the origin request. |Since `hostname` replaces the request host the value will also be used for a server identity check during a TLS handshake with the origin.|-|
| `origin`                        |string|-|URL to connect to for backend requests.|&#9888; required.  &#9888; Must start with the scheme `http://...`.|-|
| `path`                          | string|-|Changeable part of upstream URL.|-|-|
| `path_prefix`  | string|-|Prefixes all backend request paths with the given prefix|&#9888; Must start with the scheme `http://...`. |-|
| `connect_timeout`                | [duration](#duration) | `10s`      | The total timeout for dialing and connect to the origin.   |-                                   |-|
| `disable_certificate_validation` | bool               | `false`       | Disables the peer certificate validation.                                              |      - |-|
| `disable_connection_reuse`       | bool               | `false`        | Disables reusage of connections to the origin.                                          |    -  |-|
| `http2`                          | bool               | `false`         | Enables the HTTP2 support.                                                               | -    |-|
| `max_connections`                | integer                | `0` (unlimited) | The maximum number of concurrent connections in any state (_active_ or _idle_) to the origin. |-|-|
| `proxy`                          | string             | -| A proxy URL for the related origin request.      |-   | `http://SERVER-IP_OR_NAME:PORT`|
| `timeout`                        | [duration](#duration) | `300s`          | The total deadline duration a backend request has for write and read/pipe.               |-     |-|
| `ttfb_timeout`                   | [duration](#duration) | `60s`           | The duration from writing the full request to the origin and receiving the answer.        |-    |-|
| [Modifiers](#modifiers)           |- |-|All [Modifiers](#modifiers)|-|-|

#### Duration

| Duration units | Description  |
| :------------- | :----------- |
| `ns`           | nanoseconds  |
| `us` (or `µs`) | microseconds |
| `ms`           | milliseconds |
| `s`            | seconds      |
| `m`            | minutes      |
| `h`            | hours        |

### OpenAPI Block

The `openapi` block configures the backends proxy behavior to validate outgoing
and incoming requests to and from the origin. Preventing the origin from invalid
requests, and the Couper client from invalid answers. An example can be found
[here](https://github.com/avenga/couper-examples/blob/master/backend-validation/README.md).
To do so Couper uses the [OpenAPI 3 standard](https://www.openapis.org/) to load
the definitions from a given document defined with the `file` attribute.

&#9888; While ignoring request violations an invalid method or path would
lead to a non-matching _route_ which is still required for response validations.
In this case the response validation will fail if not ignored too.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`openapi`| [Backend Block](#backend-block)|-|-|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `file`                       |string|-|OpenAPI yaml definition file.|&#9888; required|`file = "openapi.yaml"`|
| `ignore_request_violations`  |bool|`false`|Log request validation results, skip error handling. |-|-|
| `ignore_response_violations` |bool|`false`|Log response validation results, skip error handling.|-|-|

### CORS Block

The `cors` block configures the CORS (Cross-Origin Resource Sharing) behavior in Couper.

<!--TODO: check if this information is correct -->
&#9888; Overrides the CORS behavior of the parent block.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`cors`|[Server Block](#server-block), [Files Block](#files-block), [SPA Block](#spa-block), [API Block](#api-block).  |no label|-|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `allowed_origins`   | list|-|A list of allowed origin(s).|Can be either of: a string with a single specific origin, `"*"` (all origins are allowed) or an array of specific origins | `allowed_origins = ["https://www.example.com", "https://www.another.host.org"]`|
| `allow_credentials` |bool|`false`| Set to `true` if the response can be shared with credentialed requests (containing `Cookie` or `Authorization` HTTP header fields).|-|-|
| `disable`           | bool|`false`|Set to `true` to disable the inheritance of CORS from the [Server Block](#server-block) in [Files Block](#files-block), [SPA Block](#spa-block) and [API Block](#api-block) contexts.|-|-|
| `max_age`           |[duration](#duration)|-|Indicates the time the information provided by the `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` response HTTP header fields.|&#9888; Can be cached|`max_age = "1h"`|

### OAuth2 CC Block

The `oauth2` block in the [Backend Block](#backend-block) context configures the OAuth2 Client Credentials flow to request a bearer token for the backend request.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`oauth2`|[Backend Block](#backend-block)|no label|[Backend Block](#backend-block)|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `backend`                       |string|-|[Backend Block Reference](#backend-block)|-|-|
| `grant_type`                    |string|-|-|&#9888; required, to be set to: `client_credentials`|`grant_type = "client_credentials"`|
| `token_endpoint`   |string|-|URL of the token endpoint at the authorization server.|&#9888; required|-|
| `client_id`|  string|-|The client identifier.|&#9888; required|-|
| `client_secret` |string|-|The client password.|&#9888; required.|-|
| `retries` |integer|`1` | The number of retries to get the token and resource, if the resource-request responds with `401 Unauthorized` HTTP status code.|-|-|
| `token_endpoint_auth_method` |string|`client_secret_basic`|Defines the method to authenticate the client at the token endpoint.|If set to `client_secret_post`, the client credentials are transported in the request body. If set to `client_secret_basic`, the client credentials are transported via Basic Authentication.|-|
| `scope`                      |string|-|  A space separated list of requested scopes for the access token.|-| `scope = "read write"` |

The HTTP header field `Accept: application/json` is automatically added to the token request. This can be modified with [request header modifiers](#request-header) in a [backend block](#backend-block).

### Websockets Block

The `websockets` block activates support for websocket connections in Couper.

| Block name | Context | Label            | Nested block(s) |
| :--------- | :------ | :--------------- | :-------------- |
| `websockets` | [Proxy Block](#proxy-block) | no label | - |

| Attribute(s) | Type | Default | Description | Characteristic(s) | Example |
| :----------- | :--- | :------ | :---------- | :---------------- | :------ |
| `timeout` | [duration](#duration) | - | The total deadline duration a websocket connection has to exists. | - | `timeout = 600s` |
| `set_request_headers` | - | - | - | Same as `set_request_headers` in [Request Header](#request-header). | - |

### Definitions Block

Use the `definitions` block to define configurations you want to reuse.

&#9888; [Access Control](#access-control) is **always** defined in the `definitions` block.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`definitions`|-|no label|[Backend Block(s)](#backend-block), [Basic Auth Block(s)](#basic-auth-block), [JWT Block(s)](#jwt-block), [JWT Signing Profile Block(s)](#jwt-signing-profile-block), [SAML Block(s)](#saml-block), [OAuth2 AC Block(s)](#oauth2-ac-block-beta), [OIDC Block(s)](#oidc-block-beta)|

<!-- TODO: add link to (still missing) example -->

### Basic Auth Block

The  `basic_auth` block lets you configure basic auth for your gateway. Like all
[Access Control](#access-control) types, the `basic_auth` block is defined in the
[Definitions Block](#definitions-block) and can be referenced in all configuration
blocks by its required _label_.

&#9888; If both `user`/`password` and `htpasswd_file` are configured, the incoming
credentials from the `Authorization` request HTTP header field are checked against
`user`/`password` if the user matches, and against the data in the file referenced
by `htpasswd_file` otherwise.

| Block name   | Context | Label | Nested block(s) |
| :----------- | :------ | :---- | :-------------- |
| `basic_auth` | [Definitions Block](#definitions-block) | &#9888; required | [Error Handler Block](ERRORS.md#error_handler-specification) |

| Attribute(s)    | Type   | Default | Description | Characteristic(s) | Example |
| :-------------- | :----- | :------ | :---------- | :---------------- | :------ |
| `user`          | string | `""`    | The user name. | - | - |
| `password`      | string | `""`    | The corresponding password. | - | - |
| `htpasswd_file` | string | `""`    | The htpasswd file. | Couper uses [Apache's httpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) file format. `apr1`, `md5` and `bcrypt` password encryptions are supported. The file is loaded once at startup. Restart Couper after you have changed it. | - |
| `realm`         | string | `""`    | The realm to be sent in a `WWW-Authenticate` response HTTP header field. | - | - |

### JWT Block

The `jwt` block lets you configure JSON Web Token access control for your gateway.
Like all [Access Control](#access-control) types, the `jwt` block is defined in
the [Definitions Block](#definitions-block) and can be referenced in all configuration blocks by its
required _label_.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
| `jwt`| [Definitions Block](#definitions-block)| &#9888; required | [JWKS `backend`](#backend-block), [Error Handler Block](ERRORS.md#error_handler-specification) |

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :-------- | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `cookie`  |string|-|Read `AccessToken` key to gain the token value from a cookie.|&#9888; available value: `AccessToken`|`cookie = "AccessToken"`|
| `header`          |string|-|-|&#9888; Implies `Bearer` if `Authorization` (case-insensitive) is used, otherwise any other header name can be used.|`header = "Authorization"` |
| `key`           |string|-|Public key (in PEM format) for `RS*` variants or the secret for `HS*` algorithm.|-|-|
| `key_file`          |string|-|Optional file reference instead of `key` usage.|-|-|
| `signature_algorithm`           |string|-|-|Valid values are: `RS256` `RS384` `RS512` `HS256` `HS384` `HS512`.|-|
| `claims`               |object|-|Object with claims that must be given for a valid token (equals comparison with JWT payload).| The claim values are evaluated per request. | `claims = { pid = request.path_params.pid }` |
| `required_claims`      |string|-|List of claim names that must be given for a valid token |-|`required_claims = ["role"]`|
| `beta_scope_claim` |string|-|name of claim specifying the scope of token|The claim value must either be a string containing a space-separated list of scope values or a list of string scope values|`beta_scope_claim = "scope"`|
| `beta_role_claim` |string|-|name of claim specifying the roles of the user represented by the token|The claim value must either be a string containing a space-separated list of role values or a list of string role values|`beta_role_claim = "role"`|
| `beta_role_map` |string|-|mapping of roles to scope values|-|`beta_role_map = { role1 = ["scope1", "scope2"], role2 = ["scope3"] }`|
| `jwks_url` | string | - | URI pointing to a set of [JSON Web Keys (RFC 7517)](https://datatracker.ietf.org/doc/html/rfc7517) | - | `jwks_url = "http://identityprovider:8080/jwks.json"` |
| `jwks_ttl` | [duration](#duration) | `"1h"` | Time period the JWK set stays valid and may be cached. | - | `jwks_ttl = "1800s"` |
| `backend`  | string| - | [backend reference](#backend-block) for enhancing JWKS requests| - | `backend = "jwks_backend"` |

If the key to verify the signatures of tokens does not change over time, it should be specified via either `key` or `key_file` (together with `signature_algorithm`).
Otherwise, a JSON web key set should be referenced via `jwks_url`; in this case, the tokens need a `kid` header.

A JWT access control configured by this block can extract scope values from
* the value of the claim specified by `beta_scope_claim` and
* the result of mapping the value of the claim specified by `beta_role_claim` using the `beta_role_map`.

The `jwt` block may also be referenced by the [`jwt_sign()` function](#functions), if it has a `signing_ttl` defined. For `HS*` algorithms the signing key is taken from `key`/`key_file`, for `RS*` algorithms, `signing_key` or `signing_key_file` have to be specified.

*Note:* A `jwt` block with `signing_ttl` cannot have the same label as a `jwt_signing_profile` block.

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :-------- | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `signing_key`       |string|-|Private key (in PEM format) for `RS*` variants.|-|-|
| `signing_key_file`  |string|-|Optional file reference instead of `signing_key` usage.|-|-|
| `signing_ttl`       |[duration](#duration)|-|The token's time-to-live (creates the `exp` claim).|-|-|

### JWT Signing Profile Block

The `jwt_signing_profile` block lets you configure a JSON Web Token signing
profile for your gateway. It is referenced in the [`jwt_sign()` function](#functions)
by its required _label_.

An example can be found
[here](https://github.com/avenga/couper-examples/blob/master/creating-jwt/README.md).

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`jwt_signing_profile`| [Definitions Block](#definitions-block)| &#9888; required |-|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `key`  |string|-|Private key (in PEM format) for `RS*` variants or the secret for `HS*` algorithm.|-|-|
| `key_file`  |string|-|Optional file reference instead of `key` usage.|-|-|
| `signature_algorithm`|-|-|-|&#9888; required. Valid values are: `RS256` `RS384` `RS512` `HS256` `HS384` `HS512`.|-|
| `ttl`  |[duration](#duration)|-|The token's time-to-live (creates the `exp` claim).|-|-|
| `claims` |object|-|Default claims for the JWT payload.| The claim values are evaluated per request. |`claims = { iss = "https://the-issuer.com" }`|
| `headers` | object | - | Additional header fields for the JWT. | `alg` and `typ` cannot be set. | `headers = { kid = "my-key-id" }` |


### OAuth2 AC Block (Beta)

The `beta_oauth2` block lets you configure the `beta_oauth_authorization_url()` [function](#functions) and an access
control for an OAuth2 **Authorization Code Grant Flow** redirect endpoint.
Like all [Access Control](#access-control) types, the `beta_oauth2` block is defined in the [Definitions Block](#definitions-block) and can be referenced in all configuration blocks by its required _label_.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`beta_oauth2`| [Definitions Block](#definitions-block)| &#9888; required | [Backend Block](#backend-block), [Error Handler Block(s)](ERRORS.md#error_handler-specification) |

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `backend`                       |string|-|[Backend Block Reference](#backend-block)| &#9888; Do not disable the peer certificate validation with `disable_certificate_validation = true`! |-|
| `authorization_endpoint` | string |-| The authorization server endpoint URL used for authorization. |&#9888; required|-|
| `token_endpoint` | string |-| The authorization server endpoint URL used for requesting the token. |&#9888; required|-|
| `token_endpoint_auth_method` |string|`client_secret_basic`|Defines the method to authenticate the client at the token endpoint.|If set to `client_secret_post`, the client credentials are transported in the request body. If set to `client_secret_basic`, the client credentials are transported via Basic Authentication.|-|
| `redirect_uri` | string |-| The Couper endpoint for receiving the authorization code. |&#9888; required. Relative URL references are resolved against the origin of the current request URL.|-|
| `grant_type` |string|-| The grant type. |&#9888; required, to be set to: `authorization_code`|`grant_type = "authorization_code"`|
| `client_id`|  string|-|The client identifier.|&#9888; required|-|
| `client_secret` |string|-|The client password.|&#9888; required.|-|
| `scope` |string|-| A space separated list of requested scopes for the access token.| - | `scope = "read write"` |
| `verifier_method` | string | - | The method to verify the integrity of the authorization code flow | &#9888; required, available values: `ccm_s256` (`code_challenge` parameter with `code_challenge_method` `S256`), `state` (`state` parameter) | `verifier_method = "ccm_s256"` |
| `verifier_value` | string or expression | - | The value of the (unhashed) verifier. | &#9888; required; e.g. using cookie value created with [`beta_oauth_verifier()` function](#functions) | `verifier_value = request.cookies.verifier` |

If the authorization server supports the `code_challenge_method` `S256` (a.k.a. PKCE, see RFC 7636), we recommend `verifier_method = "ccm_s256"`.

The HTTP header field `Accept: application/json` is automatically added to the token request. This can be modified with [request header modifiers](#request-header) in a [backend block](#backend-block).

### OIDC Block (Beta)

The `beta_oidc` block lets you configure the `beta_oauth_authorization_url()` [function](#functions) and an access
control for an OIDC **Authorization Code Grant Flow** redirect endpoint.
Like all [Access Control](#access-control) types, the `beta_oidc` block is defined in the [Definitions Block](#definitions-block) and can be referenced in all configuration blocks by its required _label_.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`beta_oidc`| [Definitions Block](#definitions-block)| &#9888; required | [Backend Block](#backend-block), [Error Handler Block(s)](ERRORS.md#error_handler-specification) |

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `backend`                       |string|-|[Backend Block Reference](#backend-block)| &#9888; Do not disable the peer certificate validation with `disable_certificate_validation = true`! |-|
| `configuration_url` | string |-| The OpenID configuration URL. |&#9888; required|-|
| `configuration_ttl` | [duration](#duration) | `1h` | The duration to cache the OpenID configuration located at `configuration_url`. | - | `configuration_ttl = "1d"` |
| `token_endpoint_auth_method` |string|`client_secret_basic`|Defines the method to authenticate the client at the token endpoint.|If set to `client_secret_post`, the client credentials are transported in the request body. If set to `client_secret_basic`, the client credentials are transported via Basic Authentication.|-|
| `redirect_uri` | string |-| The Couper endpoint for receiving the authorization code. |&#9888; required. Relative URL references are resolved against the origin of the current request URL.|-|
| `client_id`|  string|-|The client identifier.|&#9888; required|-|
| `client_secret` |string|-|The client password.|&#9888; required.|-|
| `scope` |string|-| A space separated list of requested scopes for the access token.|`openid` is automatically added.| `scope = "profile read"` |
| `verifier_method` | string | - | The method to verify the integrity of the authorization code flow | available values: `ccm_s256` (`code_challenge` parameter with `code_challenge_method` `S256`), `nonce` (`nonce` parameter) | `verifier_method = "nonce"` |
| `verifier_value` | string or expression | - | The value of the (unhashed) verifier. | &#9888; required; e.g. using cookie value created with [`beta_oauth_verifier()` function](#functions) | `verifier_value = request.cookies.verifier` |

If the OpenID server supports the `code_challenge_method` `S256` the default value for `verifier_method`is `ccm_s256`, `nonce` otherwise.

The HTTP header field `Accept: application/json` is automatically added to the token request. This can be modified with [request header modifiers](#request-header) in a [backend block](#backend-block).

### SAML Block

The `saml` block lets you configure the `saml_sso_url()` [function](#functions) and an access
control for a SAML Assertion Consumer Service (ACS) endpoint.
Like all [Access Control](#access-control) types, the `saml` block is defined in
the [Definitions Block](#definitions-block) and can be referenced in all configuration blocks by its
required _label_.

|Block name|Context|Label|Nested block(s)|
| :--------| :-----------| :-----------| :-----------|
|`saml`| [Definitions Block](#definitions-block)| &#9888; required | [Error Handler Block](ERRORS.md#error_handler-specification) |

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
|`idp_metadata_file`|string|-|File reference to the Identity Provider metadata XML file.|&#9888; required|-|
|`sp_acs_url`  |string|-|The URL of the Service Provider's ACS endpoint.|&#9888; required. Relative URL references are resolved against the origin of the current request URL.|-|
| `sp_entity_id`   |string|-|The Service Provider's entity ID.|&#9888; required|-|
| `array_attributes`|string|-|A list of assertion attributes that may have several values.|-|-|

Some information from the assertion consumed at the ACS endpoint is provided in the context at `request.context.<label>`:

- the `NameID` of the assertion's `Subject` (`request.context.<label>.sub`)
- the session expiry date `SessionNotOnOrAfter` (as UNIX timestamp: `request.context.<label>.exp`)
- the attributes (`request.context.<label>.attributes.<name>`)

### Settings Block

The `settings` block lets you configure the more basic and global behavior of your
gateway instance.

| Context | Label    | Nested block(s) |
| :------ | :------- | :-------------- |
| -       | no label | -               |

| Attribute(s)                    | Type   | Default             | Description | Characteristic(s) | Example |
| :------------------------------ | :----- | :------------------ | :---------- | :---------------- | :------ |
| `accept_forwarded_url`          | list   | `[]`                | Which `X-Forwarded-*` request headers should be accepted to change the [variables](#variables) `request.url`, `request.origin`, `request.protocol`, `request.host`, `request.port`. Valid values: `proto`, `host`, `port` |-| `["proto","host","port"]` |
| `default_port`                  | number | `8080`              | Port which will be used if not explicitly specified per host within the [`hosts`](#server-block) list. |-|-|
| `health_path`                   | string | `/healthz`          | Health path which is available for all configured server and ports. |-|-|
| `https_dev_proxy`               | list   | `[]`                | List of tls port mappings to define the tls listen port and the target one. A self-signed certificate will be generated on the fly based on given hostname. | Certificates will be hold in memory and are generated once. | `["443:8080", "8443:8080"]` |
| `log_format`                    | string | `common`            | Switch for tab/field based colored view or json log lines. |-|-|
| `log_level`                     | string | `info`              | Set the log-level to one of: `info`, `panic`, `fatal`, `error`, `warn`, `debug`, `trace`. |-|-| 
| `log_pretty`                    | bool   | `false`             | Global option for `json` log format which pretty prints with basic key coloring. |-|-|
| `no_proxy_from_env`             | bool   | `false`             | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). |-|-|
| `request_id_accept_from_header` | string |  `""`               | Name of a client request HTTP header field that transports the `request.id` which Couper takes for logging and transport to the backend (if configured). |-| `X-UID` |
| `request_id_backend_header`     | string | `Couper-Request-ID` | Name of a HTTP header field which Couper uses to transport the `request.id` to the backend. |-|-|
| `request_id_client_header`      | string | `Couper-Request-ID` | Name of a HTTP header field which Couper uses to transport the `request.id` to the client. |-|-|
| `request_id_format`             | string | `common`            | If set to `uuid4` a rfc4122 uuid is used for `request.id` and related log fields. |-|-|
| `secure_cookies`                | string | `""`                | If set to `"strip"`, the `Secure` flag is removed from all `Set-Cookie` HTTP header fields. |-|-|
| `xfh`                           | bool   | `false`             | Option to use the `X-Forwarded-Host` header as the request host. |-|-|


### Defaults Block

The `defaults` block lets you define default values.

| Block name  |Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
| `defaults`  | -| -| -|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `environment_variables` | map | – | One or more environment variable assigments|-|`environment_variables = {ORIGIN = "https://httpbin.org" ...}`|

Examples:

- [`environment_variables`](https://github.com/avenga/couper-examples/blob/master/env-var/README.md).

## Access Control

The configuration of access control is twofold in Couper: You define the particular
type (such as `jwt` or `basic_auth`) in `definitions`, each with a distinct label (must not be one of the reserved names: `scopes`).
Anywhere in the `server` block those labels can be used in the `access_control`
list to protect that block. &#9888; access rights are inherited by nested blocks.
You can also disable `access_control` for blocks. By typing `disable_access_control = ["bar"]`,
the `access_control` type `bar` will be disabled for the corresponding block context.

All access controls have an option to handle related errors. Please refer to [Errors](ERRORS.md).

## Health-Check

The health check will answer a status `200 OK` on every port with the configured
`health_path`. As soon as the gateway instance will receive a `SIGINT` or `SIGTERM`
the check will return a status `500 StatusInternalServerError`. A shutdown delay
of `5s` for example allows the server to finish all running requests and gives a load-balancer
time to pick another gateway instance. After this delay the server goes into
shutdown mode with a deadline of `5s` and no new requests will be accepted.
The shutdown timings defaults to `0` which means no delaying with development setups.
Both durations can be configured via environment variable. Please refer to the [docker document](./../DOCKER.md).

## Variables

### `couper`

| Variable                         | Type   | Description                                                                                                                                                                                                                                                                         | Example |
| :------------------------------- | :----- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------ |
| `version`                        | string | Couper's version number                                                                                                                                                                                                                                                             | `1.3.1` |

### `env`

Environment variables can be accessed everywhere within the configuration file
since these references get evaluated at start.

You may provide default values by means of `environment_variables` in the [`defaults` block](#defaults-block):

```hcl
// ...
   origin = env.ORIGIN
// ...
defaults {
  environment_variables = {
    ORIGIN = "http://localhost/"
    TIMEOUT = "3s"
  }
}
```

### `request`

| Variable                         | Type            | Description                                                                                                                                                                                                                                                                         | Example                                     |
| :------------------------------- | :-------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------ |
| `id`                             | string          | Unique request id                                                                                                                                                                                                                                                                   |                                             |
| `method`                         | string          | HTTP method                                                                                                                                                                                                                                                                         | `GET`                                       |
| `headers.<name>`                 | string          | HTTP request header value for requested lower-case key                                                                                                                                                                                                                              |                                             |
| `cookies.<name>`                 | string          | Value from `Cookie` request header for requested key (&#9888; last wins!)                                                                                                                                                                                                           |                                             |
| `query.<name>`                   | tuple of string | Query parameter values                                                                                                                                                                                                                                                              |                                             |
| `path_params.<name>`             | string          | Value from a named path parameter defined within an endpoint path label                                                                                                                                                                                                             |                                             |
| `body`                           | string          | Request message body                                                                                                                                                                                                                                                                |                                             |
| `form_body.<name>`               | tuple of string | Parameter in a `application/x-www-form-urlencoded` body                                                                                                                                                                                                                             |                                             |
| `json_body.<name>`               | various         | Access json decoded object properties. Media type must be `application/json` or `application/*+json`.                                                                                                                                                                               |                                             |
| `context.<name>.<property_name>` | various         | Request context containing information from the [Access Control](#access-control).                                                                                                                                                                                                  |                                             |
| `url`                            | string          | Request URL                                                                                                                                                                                                                                                                         | `https://www.example.com/path/to?q=val&a=1` |
| `origin`                         | string          | Origin of the request URL                                                                                                                                                                                                                                                           | `https://www.example.com`                   |
| `protocol`                       | string          | Request protocol                                                                                                                                                                                                                                                                    | `https`                                     |
| `host`                           | string          | Host of the request URL                                                                                                                                                                                                                                                             | `www.example.com`                           |
| `port`                           | integer         | Port of the request URL                                                                                                                                                                                                                                                             | `443`                                       |
| `path`                           | string          | Request URL path                                                                                                                                                                                                                                                                    | `/path/to`                                  |

The value of `context.<name>` depends on the type of block referenced by `<name>`.

For a [JWT Block](#jwt-block), the variable contains claims from the JWT used for [Access Control](#access-control).

For a [SAML Block](#saml-block), the variable contains

- `sub`: the `NameID` of the SAML assertion
- `exp`: optional expiration date (value of `SessionNotOnOrAfter` of the SAML assertion)
- `attributes`: a map of attributes from the SAML assertion

For an [OAuth2 AC Block](#oauth2-ac-block-beta), the variable contains the response from the token endpoint, e.g.

- `access_token`: the access token retrieved from the token endpoint
- `token_type`: the token type
- `expires_in`: the token lifetime
- `scope`: the granted scope (if different from the requested scope)

and for OIDC additionally:

- `id_token`: the ID token
- `id_token_claims`: a map of claims from the ID token
- `userinfo`: a map of claims retrieved from the userinfo endpoint

### `backend_requests`

`backend_requests.<label>` is a list of all backend requests, and their variables.
To access a specific request use the related label. [Request](#request-block) and
[Proxy](#proxy-block) blocks without a label will be available as `default`.
To access the HTTP method of the `default` request use `backend_requests.default.method` .

| Variable                         | Type            | Description                                                                                                                                                                                                                                                                          | Example                                     |
| :------------------------------- | :-------------- |  :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------ |
| `id`                             | string          | Unique request id                                                                                                                                                                                                                                                                    |                                             |
| `method`                         | string          | HTTP method                                                                                                                                                                                                                                                                          | `GET`                                       |
| `headers.<name>`                 | string          | HTTP request header value for requested lower-case key                                                                                                                                                                                                                               |                                             |
| `cookies.<name>`                 | string          | Value from `Cookie` request header for requested key (&#9888; last wins!)                                                                                                                                                                                                            |                                             |
| `query.<name>`                   | tuple of string | Query parameter values                                                                                                                                                                                                                                                               |                                             |
| `body`                           | string          | Backend request message body                                                                                                                                                                                                                                                         |                                             |
| `form_body.<name>`               | tuple of string | Parameter in a `application/x-www-form-urlencoded` body                                                                                                                                                                                                                              |                                             |
| `json_body.<name>`               | various         | Access json decoded object properties. Media type must be `application/json` or `application/*+json`.                                                                                                                                                                                |                                             |
| `context.<name>.<property_name>` | various         | Request context containing claims from JWT used for [Access Control](#access-control) or information from a SAML assertion, `<name>` being the [JWT Block's](#jwt-block) or [SAML Block's](#saml-block) label and `property_name` being the claim's or assertion information's name  |                                             |
| `url`                            | string          | Backend request URL                                                                                                                                                                                                                                                                  | `https://www.example.com/path/to?q=val&a=1` |
| `origin`                         | string          | Origin of the backend request URL                                                                                                                                                                                                                                                    | `https://www.example.com`                   |
| `protocol`                       | string          | Backend request protocol                                                                                                                                                                                                                                                             | `https`                                     |
| `host`                           | string          | Host of the backend request URL                                                                                                                                                                                                                                                      | `www.example.com`                           |
| `port`                           | integer         | Port of the backend request URL                                                                                                                                                                                                                                                      | `443`                                       |
| `path`                           | string          | Backend request URL path                                                                                                                                                                                                                                                             | `/path/to`                                  |

### `backend_responses`

`backend_responses.<label>` is a list of all backend responses, and their variables. Same behaviour as for `backend_requests`.
Use the related label to access a specific response.
[Request](#request-block) and [Proxy](#proxy-block) blocks without a label will be available as `default`.
To access the HTTP status code of the `default` response use `backend_responses.default.status` .

| Variable           | Type    | Description                                                                                           | Example |
| :----------------- | :------ | :---------------------------------------------------------------------------------------------------- | :------ |
| `status`           | integer | HTTP status code                                                                                      | `200` |
| `headers.<name>`   | string  | HTTP response header value for requested lower-case key                                               | |
| `cookies.<name>`   | string  | Value from `Set-Cookie` response header for requested key (&#9888; last wins!)                        | |
| `body`             | string  | The response message body                                                                             | |
| `json_body.<name>` | various | Access json decoded object properties. Media type must be `application/json` or `application/*+json`. | |

## Functions

| Name                           | Type            | Description                                                                                                                                                                                                                                                                                          | Arguments                           | Example                                              |
| :----------------------------- | :-------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :---------------------------------- | :--------------------------------------------------- |
| `base64_decode`                | string          | Decodes Base64 data, as specified in RFC 4648.                                                                                                                                                                                                                                                       | `encoded` (string)                  | `base64_decode("Zm9v")`                              |
| `base64_encode`                | string          | Encodes Base64 data, as specified in RFC 4648.                                                                                                                                                                                                                                                       | `decoded` (string)                  | `base64_encode("foo")`                               |
| `coalesce`                     |                 | Returns the first of the given arguments that is not null.                                                                                                                                                                                                                                           | `arg...` (various)                  | `coalesce(request.cookies.foo, "bar")`               |
| `json_decode`                  | various         | Parses the given JSON string and, if it is valid, returns the value it represents.                                                                                                                                                                                                                   | `encoded` (string)                  | `json_decode("{\"foo\": 1}")`                        |
| `json_encode`                  | string          | Returns a JSON serialization of the given value.                                                                                                                                                                                                                                                     | `val` (various)                     | `json_encode(request.context.myJWT)`                 |
| `jwt_sign`                     | string          | jwt_sign creates and signs a JSON Web Token (JWT) from information from a referenced [JWT Signing Profile Block](#jwt-signing-profile-block) (or [JWT Block](#jwt-block) with `signing_ttl`) and additional claims provided as a function parameter.                                                                                                 | `label` (string), `claims` (object) | `jwt_sign("myJWT")`                                  |
| `merge`                        | object or tuple | Deep-merges two or more of either objects or tuples. `null` arguments are ignored. A `null` attribute value in an object removes the previous attribute value. An attribute value with a different type than the current value is set as the new value. `merge()` with no parameters returns `null`. | `arg...` (object or tuple)          | `merge(request.headers, { x-additional = "myval" })` |
| `beta_oauth_authorization_url` | string          | Creates an OAuth2 authorization URL from a referenced [OAuth2 AC Block](#oauth2-ac-block-beta) or [OIDC Block](#oidc-block-beta).                                                                                                                                                                                                      | `label` (string)                    | `beta_oauth_authorization_url("myOAuth2")`           |
| `beta_oauth_verifier`          | string          | Creates a cryptographically random key as specified in RFC 7636, applicable for all verifier methods; e.g. to be set as a cookie and read into `verifier_value`. Multiple calls of this function in the same client request context return the same value.                                           |                                     | `beta_oauth_verifier()`                         |
| `saml_sso_url`                 | string          | Creates a SAML SingleSignOn URL (including the `SAMLRequest` parameter) from a referenced [SAML Block](#saml-block).                                                                                                                                                                                 | `label` (string)                    | `saml_sso_url("mySAML")`                             |
| `to_lower`                     | string          | Converts a given string to lowercase.                                                                                                                                                                                                                                                                | `s` (string)                        | `to_lower(request.cookies.name)`                     |
| `to_upper`                     | string          | Converts a given string to uppercase.                                                                                                                                                                                                                                                                | `s` (string)                        | `to_upper("CamelCase")`                              |
| `unixtime`                     | integer         | Retrieves the current UNIX timestamp in seconds.                                                                                                                                                                                                                                                     |                                     | `unixtime()`                                         |
| `url_encode`                   | string          | URL-encodes a given string according to RFC 3986.                                                                                                                                                                                                                                                    | `s` (string)                        | `url_encode("abc%&,123")`                            |

## Modifiers

- [Request Header](#request-header)
- [Response Header](#response-header)
- [Set Response Status](#set-response-status)
- [Query Parameter](#query-parameter)
- [Form Parameter](#form-parameter)

### Request Header

Couper offers three attributes to manipulate the request header fields. The header
attributes can be defined unordered within the configuration file but will be
executed ordered as follows:

| Modifier                 | Contexts                                                                                                                                                | Description                                                       |
| :----------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------ | :---------------------------------------------------------------- |
| `remove_request_headers` | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | list of request header to be removed from the upstream request.   |
| `set_request_headers`    | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to set request header in the upstream request. |
| `add_request_headers`    | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to add request header to the upstream request. |

All `*_request_headers` are executed from: `endpoint`, `proxy`, `backend` and `error_handler`.

### Response Header

Couper offers three attributes to manipulate the response header fields. The header
attributes can be defined unordered within the configuration file but will be
executed ordered as follows:

| Modifier                  | Contexts                                                                                                                                                                                                                                                              | Description                                                       |
| :------------------------ | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :---------------------------------------------------------------- |
| `remove_response_headers` | [Server Block](#server-block), [Files Block](#files-block), [SPA Block](#spa-block), [API Block](#api-block), [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | list of response header to be removed from the client response.   |
| `set_response_headers`    | [Server Block](#server-block), [Files Block](#files-block), [SPA Block](#spa-block), [API Block](#api-block), [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to set response header in the client response. |
| `add_response_headers`    | [Server Block](#server-block), [Files Block](#files-block), [SPA Block](#spa-block), [API Block](#api-block), [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to add response header to the client response. |

All `*_response_headers` are executed from: `server`, `files`, `spa`, `api`, `endpoint`, `proxy`, `backend` and `error_handler`.

### Set Response Status

The `set_response_status` attribute allows to modify the HTTP status code to the
given value.

| Modifier              | Contexts                                                                                            | Description                                        |
| :-------------------- | :-------------------------------------------------------------------------------------------------- | :------------------------------------------------- |
| `set_response_status` | [Endpoint Block](#endpoint-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | HTTP status code to be set to the client response. |

If the HTTP status code ist set to `204`, the reponse body and the HTTP header
field `Content-Length` is removed from the client response, and a warning is logged.

## Parameters

### Query Parameter

Couper offers three attributes to manipulate the query parameter. The query
attributes can be defined unordered within the configuration file but will be
executed ordered as follows:

| Modifier              | Contexts                                                                                                                                                | Description                                                             |
| :-------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------ | :---------------------------------------------------------------------- |
| `remove_query_params` | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | list of query parameters to be removed from the upstream request URL.   |
| `set_query_params`    | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to set query parameters in the upstream request URL. |
| `add_query_params`    | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to add query parameters to the upstream request URL. |

All `*_query_params` are executed from: `endpoint`, `proxy`, `backend` and `error_handler`.

```hcl
server "my_project" {
  api {
    endpoint "/" {
      proxy {
        backend = "example"
      }
    }
  }
}

definitions {
  backend "example" {
    origin = "http://example.com"

    remove_query_params = ["a", "b"]

    set_query_params = {
      string = "string"
      multi = ["foo", "bar"]
      "${request.headers.example}" = "yes"
    }

    add_query_params = {
      noop = request.headers.noop
      null = null
      empty = ""
    }
  }
}
```

### Form Parameter

Couper offers three attributes to manipulate the form parameter. The form
attributes can be defined unordered within the configuration file but will be
executed ordered as follows:

| Modifier             | Contexts                                                                                                                                                | Description                                                             |
| :------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------ | :---------------------------------------------------------------------- |
| `remove_form_params` | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | list of form parameters to be removed from the upstream request body.   |
| `set_form_params`    | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to set form parameters in the upstream request body. |
| `add_form_params`    | [Endpoint Block](#endpoint-block), [Proxy Block](#proxy-block), [Backend Block](#backend-block), [Error Handler](ERRORS.md#error_handler-specification) | Key/value(s) pairs to add form parameters to the upstream request body. |

All `*_form_params` are executed from: `endpoint`, `proxy`, `backend` and `error_handler`.

The `*_form_params` apply only to requests with the `POST` method and
the `Content-Type: application/x-www-form-urlencoded` HTTP header field.

```hcl
server "my_project" {
  api {
    endpoint "/" {
      proxy {
        backend = "example"
      }
    }
  }
}

definitions {
  backend "example" {
    origin = "http://example.com"

    remove_form_params = ["a", "b"]

    set_form_params = {
      string = "string"
      multi = ["foo", "bar"]
      "${request.headers.example}" = "yes"
    }

    add_form_params = {
      noop = request.headers.noop
      null = null
      empty = ""
    }
  }
}
```

### Path Parameter

An endpoint label could be defined as `endpoint "/app/{section}/{project}/view" { ... }`
to access the named path parameter `section` and `project` via `request.path_params.*`.
The values would map as following for the request path: `/app/nature/plant-a-tree/view`:

| Variable                      | Value          |
| :---------------------------- | :------------- |
| `request.path_params.section` | `nature`       |
| `request.path_params.project` | `plant-a-tree` |
