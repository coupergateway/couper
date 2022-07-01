# Basic Auth

The  `basic_auth` block lets you configure basic auth for your gateway. Like all
[Access Control](#access-control) types, the `basic_auth` block is defined in the
[Definitions Block](#definitions-block) and can be referenced in all configuration
blocks by its required _label_.

&#9888; If both `user`/`password` and `htpasswd_file` are configured, the incoming
credentials from the `Authorization` request HTTP header field are checked against
`user`/`password` if the user matches, and against the data in the file referenced
by `htpasswd_file` otherwise.

| Block name   | Context                                 | Label            | Nested block(s)                                |
|:-------------|:----------------------------------------|:-----------------|:-----------------------------------------------|
| `basic_auth` | [Definitions Block](#definitions-block) | &#9888; required | [Error Handler Block(s)](#error-handler-block) |

| Attribute(s)        | Type   | Default | Description                                                              | Characteristic(s)                                                                                                                                                                                                                                        | Example |
|:--------------------|:-------|:--------|:-------------------------------------------------------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------|
| `user`              | string | `""`    | The user name.                                                           | -                                                                                                                                                                                                                                                        | -       |
| `password`          | string | `""`    | The corresponding password.                                              | -                                                                                                                                                                                                                                                        | -       |
| `htpasswd_file`     | string | `""`    | The htpasswd file.                                                       | Couper uses [Apache's httpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) file format. `apr1`, `md5` and `bcrypt` password encryptions are supported. The file is loaded once at startup. Restart Couper after you have changed it. | -       |
| `realm`             | string | `""`    | The realm to be sent in a `WWW-Authenticate` response HTTP header field. | -                                                                                                                                                                                                                                                        | -       |
| `custom_log_fields` | object | -       | Defines log fields for [Custom Logging](LOGS.md#custom-logging).         | &#9888; Inherited by nested blocks.                                                                                                                                                                                                                      | -       |

The `user` is accessable via `request.context.<label>.user` for successfully authenticated requests.
