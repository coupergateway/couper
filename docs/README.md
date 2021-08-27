# Introduction

Couper is a lightweight open-source API gateway that acts as an entry point for clients to your application (frontend API gateway) and an exit point to upstream services (upstream API gateway).

It adds access control, observability, and back-end connectivity on a separate layer. This will keep your core application code more simple.

Couper does not need any special development skills and offers easy configuration and integration.

## Architectural Overview

![architecture](./img/architecture.png)

| Entity           | Description                                                                          |
| :--------------- | :----------------------------------------------------------------------------------- |
| Frontend         | Browser, App or API Client that sends requests to Couper.                            |
| Frontend API     | Couper acts as an entry point for clients to your application.                       |
| Backend Service  | Your core application - no matter which technology, if monolithic or micro-services. |
| Upstream API     | Couper acts as an exit point to upstream services for your application.              |
| Remote Service   | Any upstream service or system which is accessible via HTTP.                         |
| Protected System | Representing a service or system that offers protected resources via HTTP.           |

## Getting Started

Couper is available as _docker
image_ from [Docker Hub](https://hub.docker.com/r/avenga/couper)

Running Couper requires a working [Docker](https://www.docker.com/) setup on your
computer. Please visit the [get started guide](https://docs.docker.com/get-started/) to get prepared.

To download/install Couper, open a terminal and execute:

```sh
docker pull avenga/couper
```

Couper needs a configuration file to know what to do.

Create a directory with an empty `couper.hcl` file.

Copy/paste the following configuration to the file and save it.

```hcl
server "hello" {
  endpoint "/**" {
    response {
      body = "Hello World!"
    }
  }
}
```

Now `cd` into the directory with the configuration file and start Couper in a docker container:

```sh
$ docker run --rm -p 8080:8080 -v "$(pwd)":/conf avenga/couper
{"addr":"0.0.0.0:8080","level":"info","message":"couper gateway is serving","timestamp":"2020-08-27T16:39:18Z","type":"couper"}
```

Now Couper is serving on your computer's port _8080_. Point your
browser or `curl` to [`localhost:8080`](http://localhost:8080/) to see what's going on.

Press `CTRL+c` to stop the container.

-----

Please refer to the full [Configuration](reference/README.md) reference.
If you prefer to learn about Couper by checking out certain features, visit the [example repository](https://github.com/avenga/couper-examples).
