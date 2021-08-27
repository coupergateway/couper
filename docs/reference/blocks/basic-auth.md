# Basic Auth Block

The `basic_auth` block configures the basic auth. Like all
[Access Control Types](../access-control.md#access-control-types), the `basic_auth`
block is defined in the [Definitions Block](definitions.md) and can be referenced
in other [Blocks](../blocks.md) by its required `label`.

| Block name   | Label               | Related blocks                      |
| ------------ | ------------------- | ----------------------------------- |
| `basic_auth` | &#10003; (required) | [Definitions Block](definitions.md) |

## Nested blocks

* [Error Handler Block](error-handler.md)

## Attributes

| Attribute                           | Type   | Default   | Description |
| ----------------------------------- | ------ | --------- | ----------- |
| [`htpasswd_file`](../attributes.md) | string | `""`      | Couper uses [Apache's httpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) file format. `apr1`, `md5` and `bcrypt` password encryptions are supported. |
| [`password`](../attributes.md)      | string | `""`      | The corresponding password. |
| [`realm`](../attributes.md)         | string | `""`      | The realm to be sent in a `WWW-Authenticate` response HTTP header field. |
| [`user`](../attributes.md)          | string | `""`      | The user name. |

```diff
! If both "user/password" and "htpasswd_file" are configured, the incoming credentials from the "Authorization" request HTTP header field are checked against "user/password" if the user matches, and against the data in the file referenced by "htpasswd_file" otherwise.
```

```diff
! The file referenced by "htpasswd_file" is loaded once at startup. Restart Couper after You have changed it.
```

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Backend Block](backend.md)
* &#8674; [CORS Block](cors.md)
