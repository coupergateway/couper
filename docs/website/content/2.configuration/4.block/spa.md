# SPA

The `spa` blocks configure the Web serving for SPA assets. Can be defined multiple times as long as the `base_path`+`paths` is unique.

| Block name | Context                                     | Label    |
|:-----------|:--------------------------------------------|:---------|
| `spa`      | [Server Block](/configuration/block/server) | Optional |

```hcl
spa {
    base_path = "/my-app" # mounts on /my-app(/**,/special)
    bootstrap_file = "./htdocs/index.html"
    paths = ["/**", "/special"]
}
```

## Bootstrap Data

As it could get complicated to configure an SPA based on its environment (urls, clientIDs, ...) Couper can
inject those environment based values into the `bootstrap_file` before serving it to the client.

The first `bootstrap_data_placeholder` will be replaced with the evaluated value of `bootstrap_data`.
This happens on startup and is meant to inject `env` values.

### `bootstrap_data` Example

```hcl
# couper.hcl
spa {
    bootstrap_file = "./htdocs/index.html"
    paths = ["/**"]
    bootstrap_data = {
      url: env.MY_API_URL,
      prop: "value",
    }
}
```

```html
<!-- ./htdocs/index.html -->
<!DOCTYPE html>
<html lang="en">
  <head>
    <script>
      try {
        window.AppConfig = __BOOTSTRAP_DATA__;
      } catch(e) {
        console.warn('DEVELOPMENT MODE: ', e)
        window.AppConfig = {} // fallback for local development
      }
    </script>
  </head>
  <body>App</body>
</html>
```

The result would be the following snippet. Just place this `<script>` tag before all other script references in the head tag.
Then the application can access the `window.AppConfig` object to bootstrap its configuration.

```html
<!-- ... -->
  <script>
    try {
      window.AppConfig = {
        "url": "https://docs.couper.io/",
        "prop": "value",
      };
    } catch(e) {
      console.warn('DEVELOPMENT MODE: ', e)
      window.AppConfig = {} // fallback for local development
    }
  </script>
<!-- ... -->
```

::attributes
---
values: [
  {
    "default": "[]",
    "description": "Sets predefined [access control](../access-control) for `spa` block context.",
    "name": "access_control",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to add as response headers in the client response.",
    "name": "add_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Configures the path prefix for all requests.",
    "name": "base_path",
    "type": "string"
  },
  {
    "default": "",
    "description": "JSON object which replaces the placeholder from `bootstrap_file` content.",
    "name": "bootstrap_data",
    "type": "object"
  },
  {
    "default": "\"__BOOTSTRAP_DATA__\"",
    "description": "String which will be replaced with `bootstrap_data`.",
    "name": "bootstrap_data_placeholder",
    "type": "string"
  },
  {
    "default": "",
    "description": "Location of the bootstrap file.",
    "name": "bootstrap_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "Disables access controls by name.",
    "name": "disable_access_control",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "List of SPA paths that need the bootstrap file.",
    "name": "paths",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "List of names to remove headers from the client response.",
    "name": "remove_response_headers",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to set as response headers in the client response.",
    "name": "set_response_headers",
    "type": "object"
  }
]

---
::

::blocks
---
values: [
  {
    "description": "Configures [CORS](/configuration/block/cors) settings (zero or one).",
    "name": "cors"
  }
]

---
::
