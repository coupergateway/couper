# Couper 2

Couper 2 is designed to support developers building and operating API-driven Web projects by offering security and observability functionality in a  lightweight API gateway component.

* Tutorials: [couper examples](https://github.com/avenga/couper-examples)

**Couper**
* is a proxy component connecting clients with (micro) services
* adds security and observability to the project 
* needs no special development skills
* is easy to configure & integrate
* is [Avenga’s](https://www.avenga.com/) standard technology

Couper provides some key features:

- An **easy configuration** powered by [HCL 2](https://github.com/hashicorp/hcl/tree/hcl2)
- Exposes local and remote backend services in a consolidated frontend API
- Operation and **observability**:
    - Timeout handling
    - Logging access and upstream requests as tab fields or json format
    - Metrics endpoint (soon)
    - Health probes for backends (soon)
- Centralized **Access-Control** layer:
    - Basic-Auth (soon)
    - JWT
        - RS/HM 256,386,512 algorithms
        - Custom claim validation
        - pass claim values to upstream services
- Uniform **error handling** with custom templates across different backend services
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
