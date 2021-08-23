# Couper

![Go](https://github.com/avenga/couper/workflows/Go/badge.svg)
[![Go Report](https://goreportcard.com/badge/github.com/avenga/couper)](https://goreportcard.com/report/github.com/avenga/couper)
![Docker](https://github.com/avenga/couper/workflows/Docker/badge.svg)

![Couper](docs/img/couper-logo.svg)

**Couper** is a lightweight API gateway designed to support developers in building and operating API-driven Web projects.

## Getting started

* The quickest way to start is to use our [Docker image](https://hub.docker.com/r/avenga/couper).
* The [documentation](https://github.com/avenga/couper/tree/master/docs) gives an introduction to Couper.
* Check out the [example repository](https://github.com/avenga/couper-examples) to learn about Couper's features in detail.
* Dive into the [Configuration Reference](docs/REFERENCE.md)
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
* **Configurable Service Connectivity**
* Upstream Validation & CORS
* SPA & Web Serving
* Error Handling
* Observability

The full list of features of Couper 1.0 is [here](FEATURES.md) or at [https://couper.io/features](https://couper.io/features).

## Developers

*Developers* requiring [Go](https://golang.org/) to start with `make build`.
Couper requires a configuration file. You can start with a simple one and use:

```console
./couper run -f public/couper.hcl
```

## Contributing

Thanks for your interest in contributing.
If you have any questions or feedback you are welcome to start a [discussion](https://github.com/avenga/couper/discussions).
If you have an issue please open an [issue](https://github.com/avenga/couper/issues).
