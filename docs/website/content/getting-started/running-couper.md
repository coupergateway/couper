---
title: 'Running Couper'
weight: 2
slug: 'running-couper'
---

# Running Couper

Couper is available as _docker
image_ from [Docker Hub](https://hub.docker.com/r/coupergateway/couper)

Running Couper requires a working [Docker](https://www.docker.com/) setup on your
computer. Please visit the [get started guide](https://docs.docker.com/get-started/) to get prepared.

To download/install Couper, open a terminal and execute:

```sh
$ docker pull coupergateway/couper
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
$ docker run --rm -p 8080:8080 -v "$(pwd)":/conf coupergateway/couper
{"addr":"0.0.0.0:8080","level":"info","message":"couper gateway is serving","timestamp":"2020-08-27T16:39:18Z","type":"couper"}
```

Now Couper is serving on your computer's port _8080_. Point your
browser or `curl` to [`localhost:8080`](http://localhost:8080/) to see what's going on.

Press <kbd>Ctrl</kbd> + <kbd>C</kbd> to stop the container.

The [following section](/configuration/configuration-file) will give you an introduction into Couper's configuration file.

If you prefer to learn about Couper by checking out certain features, visit the [example repository](https://github.com/coupergateway/couper-examples).
