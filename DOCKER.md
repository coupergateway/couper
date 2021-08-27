# Couper

![Couper](https://raw.githubusercontent.com/avenga/couper/master/docs/img/couper-logo.svg)

Couper is designed to support developers building and operating API-driven Web projects by offering security and observability functionality in a frontend gateway component.

_For additional information, tutorials and documentation please visit the [couper repository](https://github.com/avenga/couper)._

## Usage

Couper requires a [configuration file](https://github.com/avenga/couper/tree/master/docs#conf_file) which have to be provided on start.
See our [documentation](https://github.com/avenga/couper/tree/master/docs) how to configure _couper_.

This image contains a basic configuration to serve files from `/htdocs` directory.

```sh
docker run --rm -p 8080:8080 -v `pwd`:/htdocs avenga/couper
```

## Command

The entrypoint of the image is the `/couper` binary. The command is `run`.

Therefore `docker run avenga/couper` runs `/couper run`. You could also use other commands directly:

```sh
docker run avenga/couper version
docker run avenga/couper run -watch -p 8081
```

## Environment options

Please refer to [Environment](docs/reference/environment.md) section of the
[Configuration Reference](docs/reference/README.md).
