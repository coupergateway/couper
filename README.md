# Couper

[![Go Test](https://github.com/coupergateway/couper/actions/workflows/go.yml/badge.svg)](https://github.com/coupergateway/couper/actions/workflows/go.yml)
[![Go Report](https://goreportcard.com/badge/github.com/coupergateway/couper)](https://goreportcard.com/report/github.com/coupergateway/couper)
![Docker](https://github.com/coupergateway/couper/workflows/Docker/badge.svg)
[![Code Coverage](https://qlty.sh/gh/coupergateway/projects/couper/coverage.svg)](https://qlty.sh/gh/coupergateway/projects/couper)

![Couper](docs/img/couper-logo.svg)

**Couper** is a lightweight API gateway designed to support developers in building and operating API-driven Web projects.

## Getting started

* The quickest way to start is to use our [Docker image](https://hub.docker.com/r/coupergateway/couper).
  * or via [homebrew](https://brew.sh/): `brew tap coupergateway/couper && brew install couper`
  * or via Go: `go install github.com/coupergateway/couper@latest`
  * or via [devcontainer feature](https://github.com/coupergateway/features):
    ```json
    "features": {
        "ghcr.io/coupergateway/features/couper": {}
    }
    ```
* The [documentation](https://docs.couper.io/) gives an introduction to Couper.
* Check out the [example repository](https://github.com/coupergateway/couper-examples) to learn about Couper's features in detail.
* Use-cases can be found on [couper.io](https://couper.io).

## Features

Couper …

* is a proxy component connecting clients with (micro) services
* adds access control and observability to the project
* needs no special development skills
* is easy to configure & integrate
* runs on Linux, macOS, Windows, Docker and Kubernetes.

Key features are:

* **Easy Configuration & Deployment**
* HTTP Request Routing / Forwarding
* Custom Requests and Responses
* Request / Response Manipulation
* Sequence and Parallel Backend Requests
* WebSockets Support
* Centralized **Access-Control** Layer:
  * Basic-Auth
  * JWT Validation & Signing
  * Single Sign On with SAML2
  * OAuth2 Client Credentials
  * OpenID Connect
* **Configurable Service Connectivity**
* Upstream Validation & CORS
* SPA & Web Serving
  * Inject server data / environment variables to your SPA
* Error Handling
* Observability
  * Prometheus exporter
* **Security**
  * mTLS support (server and backend)
  * Request size limiting

The full list of features of Couper 1.x is [here](FEATURES.md) or at [couper.io](https://couper.io/en/features).

## Developers

Development requires [Go](https://golang.org/). Start with `make build`.
Couper requires a [configuration file](./docs/README.md#configuration-file). You can start with a simple one and use:

```ps
./couper run -f public/couper.hcl
```

## Contributing

Thanks for your interest in contributing.

If you have any questions or feedback you are welcome to start a [discussion](https://github.com/coupergateway/couper/discussions).

If you have an issue please open an [issue](https://github.com/coupergateway/couper/issues).
