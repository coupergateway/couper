# Basic Auth

| Block name   | Context                                 | Label    | Nested block(s)                                |
|:-------------|:----------------------------------------|:---------|:-----------------------------------------------|
| `basic_auth` | [Definitions Block](/configuration/block/definitions) | required | [Error Handler Block(s)](/configuration/block/error_handler) |

The  `basic_auth` block lets you configure basic auth for your gateway. Like all
[access control](/configuration/access-control) types, the `basic_auth` block is defined in the
[`definitions` block](/configuration/block/definitions) and can be referenced in all configuration
blocks by its required _label_.

If both `user`/`password` and `htpasswd_file` are configured, the incoming
credentials from the `Authorization` request HTTP header field are checked against
`user`/`password` if the user matches, and against the data in the file referenced
by `htpasswd_file` otherwise.

The `user` is accessible via `request.context.<label>.user` variable for successfully authenticated requests.

### htpasswd_file

Couper uses [Apache's httpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) file format. `apr1`, `md5` and `bcrypt` password encryption are supported. The file is loaded once at startup. Restart Couper after you have changed it.

::attributes
---
values: [
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "The htpasswd file.",
    "name": "htpasswd_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "The corresponding password.",
    "name": "password",
    "type": "string"
  },
  {
    "default": "",
    "description": "The realm to be sent in a WWW-Authenticate response HTTP header field.",
    "name": "realm",
    "type": "string"
  },
  {
    "default": "",
    "description": "The user name.",
    "name": "user",
    "type": "string"
  }
]

---
::

::blocks
---
values: [
  {
    "description": "Configures an [error handler](/configuration/block/error_handler).",
    "name": "error_handler"
  }
]

---
::
