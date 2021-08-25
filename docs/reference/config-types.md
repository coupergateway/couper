# Configuration Reference ~ Configuration Types

* [Duration](#duration)
* [List](#list)
* [Map](#map)
* [Tuple](#tuple)

## Duration

Type `duration` represents time-based units.

| Duration unit  | Description  |
| -------------- | ------------ |
| `ns`           | nanoseconds  |
| `us` (or `µs`) | microseconds |
| `ms`           | milliseconds |
| `s`            | seconds      |
| `m`            | minutes      |
| `h`            | hours        |
| `d`            | days         |

**Example:**

```hcl
timeout = "60s"
```

## List

Type `list` represents an `array` / `list` / `slice` of parameters of the same type.

**Example:**

```hcl
access_control = ["ba", "jwt", "saml"]
```

## Map

Type `map` represents an attribute block of key/value pairs.

**Example:**

```hcl
add_request_headers = {
  Cache-Control = "public, max-age=60"
}
```

## Size

Type `size` represents size-based units.

| Size unit | Description |
| --------- | ----------- |
| `KiB`     | kilobyte    |
| `MiB`     | megabyte    |
| `GiB`     | gigabyte    |

**Example:**

```hcl
request_body_limit = "64MiB"
```

## Tuple

Like [`list`](#list) (see above), but the parameters must not have the same type.

**Example:**

```hcl
merge([1, null, "text"], [2, "test", true])
      \               /  \               /
       \             /    \             /
        \           /      \           /
         \         /        \         /
          \       /          \       /
            tuple              tuple
```

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Command Line Interface](cli.md)
* &#8674; [Environment](environment.md)
