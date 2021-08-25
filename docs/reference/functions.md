# Configuration Reference ~ Functions

Click on the function name to see more details.

| Function                                                                    | Description |
| --------------------------------------------------------------------------- | ----------- |
| [`base64_decode`](functions/base64-decode.md)                               | Decodes Base64 data as specified in [RFC 4648](https://datatracker.ietf.org/doc/html/rfc4648). |
| [`base64_encode`](functions/base64-encode.md)                               | Encodes Base64 data as specified in [RFC 4648](https://datatracker.ietf.org/doc/html/rfc4648). |
| [`beta_oauth_authorization_url`](functions/beta-oauth-authorization-url.md) | Creates an OAuth 2.0 authorization URL from a referenced [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta) or [OIDC Block](blocks/beta-oidc.md) (Beta). |
| [`beta_oauth_verifier`](functions/beta-oauth-verifier.md)                   | Creates a cryptographically random key as specified in [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636), applicable for all verifier methods; e.g. to be set as a cookie and read into `verifier_value`. Multiple calls of this function in the same client request context return the same value. |
| [`coalesce`](functions/coalesce.md)                                         | Returns the first of the given arguments that is not null. |
| [`json_decode`](functions/json-decode.md)                                   | Parses the given [JSON](https://www.json.org) string and, if it is valid, returns the value it represents. |
| [`json_encode`](functions/json-encode.md)                                   | Returns a [JSON](https://www.json.org) serialization of the given value. |
| [`jwt_sign`](functions/jwt-sign.md)                                         | Creates and signs a JSON Web Token (JWT) from information from a referenced [JWT Signing Profile Block](blocks/jwt-signing-profile.md) and additional claims provided as a function parameter. |
| [`merge`](functions/merge.md)                                               | Deep-merges two or more of either objects or [tuples](config-types.md#tuple). |
| [`saml_sso_url`](functions/saml-sso-url.md)                                 | Creates a SAML SingleSignOn URL (including the `SAMLRequest` parameter) from a referenced [SAML Block](blocks/saml.md). |
| [`to_lower`](functions/to-lower.md)                                         | Converts a given string to lowercase. |
| [`to_upper`](functions/to-upper.md)                                         | Converts a given string to uppercase. |
| [`unixtime`](functions/unixtime.md)                                         | Retrieves the current UNIX timestamp in seconds. |
| [`url_encode`](functions/url-encode.md)                                     | URL-encodes a given string according to [RFC 3986](https://datatracker.ietf.org/doc/html/rfc3986). |

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Examples](examples.md)
* &#8674; [Health-Check](health-check.md)
