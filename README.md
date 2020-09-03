# Couper

[couper.io](https://couper.io) offers *the* gateway solution to serve Single Page Applications and API's. Providing access-control and observability out of the box configured with ease.

* Tutorials: [couper examples](https://github.com/avenga/couper-examples)

Couper provides some key features:

- An **easy configuration** powered by [HCL 2](https://github.com/hashicorp/hcl/tree/hcl2)
- Operation and **observability**:
    - Timeout handling
    - Logging access and upstream requests as tab fields or json format
    - Metrics endpoint (soon)
    - Health probes for backends (soon)
- **Access-Control**:
    - Basic-Auth (soon)
    - JWT
        - RS/HM 256,386,512 algorithms
        - Custom claim validation
- Error handling with custom templates
    - api endpoints serving a json error response, defaults to html otherwise
- **SPA** support with configurable path fallbacks


Couper runs on Linux, Mac OS X, FreeBSD and Windows.

## Quick Start

* Docker image: https://hub.docker.com/r/avenga/couper
* Binary installation: https://github.com/avenga/couper/releases

## Documentation

Docs comming soon. Please take a look at our [examples](https://github.com/avenga/couper-examples).

## Contributing

Thanks for your interest in contributing. If you have an issue, or a question about *anything* please open an [issue](https://github.com/avenga/couper/issues).
