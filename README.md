# Couper

![Go](https://github.com/avenga/couper/workflows/Go/badge.svg)
![Docker](https://github.com/avenga/couper/workflows/Docker/badge.svg)

Couper is designed to support developers building and operating API-driven Web
projects by offering security and observability functionality in a frontend gateway
component.

* [Tutorials](https://github.com/avenga/couper-examples)
* [Documentation](https://github.com/avenga/couper/tree/master/docs)

## Couper

* is a proxy component connecting clients with (micro) services
* adds security and observability to the project
* needs no special development skills
* is easy to configure & integrate
* runs on Linux, Mac OS X, FreeBSD and Windows.

## Key features

* An **easy configuration** powered by [HCL 2](https://github.com/hashicorp/hcl/tree/hcl2)
* Exposes local and remote backend services in a consolidated frontend API
* Operation and **observability**:
  * Timeout handling
  * Logging access and upstream requests as tab fields or json format
  * Metrics endpoint (soon)
  * Health probes for backends (soon)
* Centralized **Access-Control** layer:
  * Basic-Auth
  * JWT
    * RS/HS 256,386,512 algorithms
    * Custom claim validation
    * pass claim values to upstream services
* Uniform **error handling** with custom templates across different backend services
  * api endpoints serving a json error response, defaults to html otherwise
  * **SPA** support with configurable path fallbacks

## Quick Start

* [Docker image](https://hub.docker.com/r/avenga/couper)
* [Binary installation](https://github.com/avenga/couper/releases)

## Contributing

Thanks for your interest in contributing.
If you have any questions or feedback you are welcome to start a [discussion](https://github.com/avenga/couper/discussions).
If you have an issue please open an [issue](https://github.com/avenga/couper/issues).
