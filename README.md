# Couper

![Go](https://github.com/avenga/couper/workflows/Go/badge.svg)
[![Go Report](https://goreportcard.com/badge/github.com/avenga/couper)](https://goreportcard.com/report/github.com/avenga/couper)
![Docker](https://github.com/avenga/couper/workflows/Docker/badge.svg)

![Couper](docs/couper-logo.svg)

**Couper** is a lightweight API gateway designed to support developers in building and operating API-driven Web projects.

## Getting started

* Check out our [example repository](https://github.com/avenga/couper-examples) for a first glance.
* Read more about our use-cases on [couper.io](https://couper.io).
* The quickest way to start is to use our [Docker image](https://hub.docker.com/r/avenga/couper).
* Continue with the [documentation](https://github.com/avenga/couper/tree/master/docs).


## Features

Couper …

* is a proxy component connecting clients with (micro) services
* adds access control and observability to the project
* needs no special development skills
* is easy to configure & integrate
* runs on Linux, Mac OS X, Windows, Docker and Kubernetes.

Key features are:

* **Easy configuration** powered by [HCL 2](https://github.com/hashicorp/hcl/tree/hcl2)
* Exposes local and remote backend services in a consolidated frontend API
* Operation and **observability**:
  * Timeout handling
  * Logging access and upstream requests as tab fields or json format
* Centralized **Access-Control** layer:
  * Basic-Auth
  * JWT
    * RS/HS 256,386,512 algorithms
    * Custom claim validation
    * pass claim values to upstream services
   * Single Sign On with SAML2
* Uniform **error handling** with custom templates across different backend services
  * api endpoints serving a json error response, defaults to html otherwise
* **SPA** support with configurable path fallbacks

The full list of features of Couper 1.0 is [here](FEATURES.md).


## Contributing

Thanks for your interest in contributing.
If you have any questions or feedback you are welcome to start a [discussion](https://github.com/avenga/couper/discussions).
If you have an issue please open an [issue](https://github.com/avenga/couper/issues).
