# Couper

[![Go Test](https://github.com/coupergateway/couper/actions/workflows/go.yml/badge.svg)](https://github.com/coupergateway/couper/actions/workflows/go.yml)
[![Go Report](https://goreportcard.com/badge/github.com/coupergateway/couper)](https://goreportcard.com/report/github.com/coupergateway/couper)
![Docker](https://github.com/coupergateway/couper/workflows/Docker/badge.svg)

![Couper](docs/website/public/img/couper-logo.svg)

**Couper** is a lightweight API gateway designed to support developers in building and operating API-driven Web projects.

## Getting started

* The quickest way to start is to use our [Docker image](https://hub.docker.com/r/coupergateway/couper).
  * or via [homebrew](https://brew.sh/): `brew tap coupergateway/couper && brew install couper`
* The [documentation](https://docs.couper.io/) gives an introduction to Couper.
* Check out the [example repository](https://github.com/coupergateway/couper-examples) to learn about Couper's features in detail.
* Use-cases can be found on [couper.io](https://couper.io).

## Features

Couper …

* is a proxy component connecting clients with (micro) services
* adds access control and observability to the project
* needs no special development skills
* is easy to configure & integrate
* runs on Linux, Mac OS X, Windows, Docker and Kubernetes.

Key features are:

* **Easy Configuration & Deployment**
* HTTP Request Routing / Forwarding
* Custom Requests and Responses
* Request / Response Manipulation
* Centralized **Access-Control** Layer:
  * Basic-Auth
  * JWT  Validation & Signing
  * Single Sign On with SAML2
  * OAuth2 Client Credentials
  * OpenID-connect
* **Configurable Service Connectivity**
* Upstream Validation & CORS
* SPA & Web Serving
  * inject server data / environment variables to your SPA
* Error Handling
* Observability
  * Prometheus exporter
* Security with mTLS support as server and for backend services

The full list of features of Couper 1.x is [here](FEATURES.md) or at [couper.io](https://couper.io/en/features).

## Developers

*Developers* requiring [Go](https://golang.org/) to start with `make build`.
Couper requires a [configuration file](./docs/README.md#configuration-file). You can start with a simple one and use:

```ps
./couper run -f public/couper.hcl
```

## Contributing

Thanks for your interest in contributing.

If you have any questions or feedback you are welcome to start a [discussion](https://github.com/coupergateway/couper/discussions).

If you have an issue please open an [issue](https://github.com/coupergateway/couper/issues).
