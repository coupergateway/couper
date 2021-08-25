# Configuration Reference ~ Attributes

In contrast to [Blocks](blocks.md), Attributes in Couper configuration always have
an [Equals (`=`)](https://en.wikipedia.org/wiki/Equals_sign) sign before the value,
even if the value sometimes looks like a [Block](blocks.md):

```hcl
// Blocks:
block {...}
block_labled "label" {...}

// Attributes:
attribute = "value"
attribute_map = {...}
```

## Attributes

| Attribute                 | Type                         | Default     | Related blocks |
| ------------------------- | ---------------------------- | ----------- | -------------- |
| `access_control`          | [list](config-types.md#list) | `[]`        | [API Block](blocks/api.md), [Endpoint Block](blocks/endpoint.md), [Files Block](blocks/files.md), [Server Block](blocks/server.md), [SPA Block](blocks/spa.md) |
| `accept_forwarded_url`    | [list](config-types.md#list) | `[]`        | [Settings Block](blocks/settings.md) |
| `add_form_params`         | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `add_query_params`        | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `add_request_headers`     | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md), [Websockets Block](blocks/websockets.md) |
| `add_response_headers`    | [map](config-types.md#map)   | `{}`        | [API Block](blocks/api.md), [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Files Block](blocks/files.md), [Proxy Block](blocks/proxy.md), [Server Block](blocks/server.md), [SPA Block](blocks/spa.md), [Websockets Block](blocks/websockets.md) |
| `allow_credentials`       | bool                         | `false`     | [CORS Block](blocks/cors.md) |
| `allowed_origins` | [list](config-types.md#list) or string| &#10005;   | [CORS Block](blocks/cors.md) |
| `array_attributes`        | [list](config-types.md#list) | `[]`        | [SAML Block](blocks/saml.md) |
| `authorization_endpoint`  | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta) |
| `backend`                 | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](blocks/oauth2-cc.md), [OIDC Block](blocks/beta-oidc.md) (Beta), [Proxy Block](blocks/proxy.md), [Request Block](blocks/request.md) |
| `base_path`               | string                       | `""`        | [API Block](blocks/api.md), [Files Block](blocks/files.md), [Server Block](blocks/server.md), [SPA Block](blocks/spa.md) |
| `basic_auth`              | string                       | `""`        | [Backend Block](blocks/backend.md) |
| `body`                    | string                       | `""`        | [Request Block](blocks/request.md), [Response Block](blocks/response.md) |
| `bootstrap_file`          | string                       | `""`        | [SPA Block](blocks/spa.md) |
| `claims`                  | [map](config-types.md#map)   | `{}`        | [JWT Block](blocks/jwt.md), [JWT Signing Profile Block](blocks/jwt-signing-profile.md) |
| `client_id`               | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](blocks/oauth2-cc.md), [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `client_secret`           | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](blocks/oauth2-cc.md), [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `configuration_url`       | string                       | `""`        | [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `configuration_ttl`  | [duration](config-types.md#duration) | &#10005; | [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `cookie`                  | string                       | `""`        | [JWT Block](blocks/jwt.md) |
| `connect_timeout`    | [duration](config-types.md#duration) | `"10s"`  | [Backend Block](blocks/backend.md) |
| `default_port`            | integer                      | `8080`      | [Settings Block](blocks/settings.md) |
| `disable`                 | bool                         | `false`     | [CORS Block](blocks/cors.md) |
| `disable_access_control`  | [list](config-types.md#list) | `[]`        | [API Block](blocks/api.md), [Endpoint Block](blocks/endpoint.md), [Files Block](blocks/files.md), [Server Block](blocks/server.md), [SPA Block](blocks/spa.md) |
| `disable_certificate_validation` | bool                  | `false`     | [Backend Block](blocks/backend.md) |
| `disable_connection_reuse`       | bool                  | `false`     | [Backend Block](blocks/backend.md) |
| `document_root`           | string                       | `""`        | [Files Block](blocks/files.md) |
| `environment_variables`   | [map](config-types.md#map)   | `{}`        | [Defaults Block](blocks/defaults.md) |
| `error_file`              | string                       | `""`        | [API Block](blocks/api.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Files Block](blocks/files.md), [Server Block](blocks/server.md) |
| `file`                    | string                       | `""`        | [OpenAPI Block](blocks/openapi.md) |
| `form_body`               | [map](config-types.md#map)   | `{}`        | [Request Block](blocks/request.md) |
| `grant_type`              | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](blocks/oauth2-cc.md) |
| `header`                  | string                       | `""`        | [JWT Block](blocks/jwt.md) |
| `headers`                 | [map](config-types.md#map)   | `{}`        | [Request Block](blocks/request.md), [Response Block](blocks/response.md) |
| `health_path`             | string                       | `"/healthz"` | [Settings Block](blocks/settings.md) |
| `hostname`                | string                       | `""`        | [Backend Block](blocks/backend.md) |
| `hosts`                   | [list](config-types.md#list) | `[":8080"]` | [Server Block](blocks/server.md) |
| `htpasswd_file`           | string                       | `""`        | [Basic Auth Block](blocks/basic-auth.md) |
| `http2`                   | bool                         | `false`     | [Backend Block](blocks/backend.md) |
| `https_dev_proxy`         | [list](config-types.md#list) | `[]`        | [Settings Block](blocks/settings.md) |
| `idp_metadata_file`       | string                       | `""`        | [SAML Block](blocks/saml.md) |
| `ignore_request_violations`  | bool                      | `false`     | [OpenAPI Block](blocks/openapi.md) |
| `ignore_response_violations` | bool                      | `false`     | [OpenAPI Block](blocks/openapi.md) |
| `json_body`               | various                      | &#10005;    | [Request Block](blocks/request.md), [Response Block](blocks/response.md) |
| `key`                     | string                       | `""`        | [JWT Block](blocks/jwt.md), [JWT Signing Profile Block](blocks/jwt-signing-profile.md) |
| `key_file`                | string                       | `""`        | [JWT Block](blocks/jwt.md), [JWT Signing Profile Block](blocks/jwt-signing-profile.md) |
| `log_format`              | string                       | `"common"`  | [Settings Block](blocks/settings.md) |
| `log_pretty`              | bool                         | `false`     | [Settings Block](blocks/settings.md) |
| `max_age`         | [duration](config-types.md#duration) | &#10005;    | [CORS Block](blocks/cors.md) |
| `max_connections`         | integer                      | &#10005;    | [Backend Block](blocks/backend.md) |
| `method`                  | string                       | `"GET"`     | [Request Block](blocks/request.md) |
| `no_proxy_from_env`       | bool                         | `false`     | [Settings Block](blocks/settings.md) |
| `origin`                  | string                       | `""`        | [Backend Block](blocks/backend.md) |
| `password`                | string                       | `""`        | [Basic Auth Block](blocks/basic-auth.md) |
| `path`                    | string                       | `""`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `path_prefix`             | string                       | `""`        | [Backend Block](blocks/backend.md) |
| `paths`                   | [list](config-types.md#list) | `[]`        | [SPA Block](blocks/spa.md) |
| `proxy`                   | string                       | `""`        | [Backend Block](blocks/backend.md) |
| `query_params`            | [map](config-types.md#map)   | `{}`        | [Request Block](blocks/request.md) |
| `realm`                   | string                       | `""`        | [Basic Auth Block](blocks/basic-auth.md) |
| `redirect_uri`            | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `remove_form_params`      | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `remove_query_params`     | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `remove_request_headers`  | [list](config-types.md#list) | `[]`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md), [Websockets Block](blocks/websockets.md) |
| `remove_response_headers` | [list](config-types.md#list) | `[]`        | [API Block](blocks/api.md), [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Files Block](blocks/files.md), [Proxy Block](blocks/proxy.md), [Server Block](blocks/server.md), [SPA Block](blocks/spa.md), [Websockets Block](blocks/websockets.md) |
| `request_body_limit`      | [size](config-types.md#size) | `"64MiB"`   | [Endpoint Block](blocks/endpoint.md) |
| `request_id_accept_from_header` | string                 | `""`        | [Settings Block](blocks/settings.md) |
| `request_id_backend_header`     | string | `"Couper-Request-ID"`       | [Settings Block](blocks/settings.md) |
| `request_id_client_header`      | string | `"Couper-Request-ID"`       | [Settings Block](blocks/settings.md) |
| `request_id_format`             | string | `"common"`                  | [Settings Block](blocks/settings.md) |
| `required_claims`         | [list](config-types.md#list) | `[]`        | [JWT Block](blocks/jwt.md) |
| `retries`                       | integer                | `1`         | [OAuth2 CC Block](blocks/oauth2-cc.md) |
| `scope`                   | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](blocks/oauth2-cc.md), [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `secure_cookies`          | string                       | `""`        | [Settings Block](blocks/settings.md) |
| `set_form_params`         | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `set_query_params`        | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `set_request_headers`     | [map](config-types.md#map)   | `{}`        | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Proxy Block](blocks/proxy.md) |
| `set_response_headers`    | [map](config-types.md#map)   | `{}`        | [API Block](blocks/api.md), [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md), [Files Block](blocks/files.md), [Proxy Block](blocks/proxy.md), [Server Block](blocks/server.md), [SPA Block](blocks/spa.md) |
| `set_response_status`     | integer                      | &#10005;    | [Backend Block](blocks/backend.md), [Endpoint Block](blocks/endpoint.md), [Error Handler Block](blocks/error-handler.md) |
| `signature_algorithm`     | string                       | `""`        | [JWT Block](blocks/jwt.md), [JWT Signing Profile Block](blocks/jwt-signing-profile.md) |
| `sp_acs_url`              | string                       | `""`        | [SAML Block](blocks/saml.md) |
| `sp_entity_id`            | string                       | `""`        | [SAML Block](blocks/saml.md) |
| `status`                  | integer                      | `200`       | [Response Block](blocks/response.md) |
| `timeout`         | [duration](config-types.md#duration) | `"300s"`    | [Backend Block](blocks/backend.md), [Websockets Block](blocks/websockets.md) |
| `token_endpoint`          | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](blocks/oauth2-cc.md) |
| `token_endpoint_auth_method` | string | `"client_secret_basic"`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](blocks/oauth2-cc.md), [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `ttfb_timeout`    | [duration](config-types.md#duration) | `"60s"`     | [Backend Block](blocks/backend.md) |
| `ttl`             | [duration](config-types.md#duration) | &#10005;    | [JWT Signing Profile Block](blocks/jwt-signing-profile.md) |
| `url`                     | string                       | `""`        | [Proxy Block](blocks/proxy.md), [Request Block](blocks/request.md) |
| `user`                    | string                       | `""`        | [Basic Auth Block](blocks/basic-auth.md) |
| `verifier_method`         | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `verifier_value`          | string                       | `""`        | [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta), [OIDC Block](blocks/beta-oidc.md) (Beta) |
| `websockets`              | bool                         | `false`     | [Proxy Block](blocks/proxy.md) |
| `xfh`                     | bool                         | `false`     | [Settings Block](blocks/settings.md) |

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Access Control](access-control.md)
* &#8674; [Beta Features](beta-features.md)
