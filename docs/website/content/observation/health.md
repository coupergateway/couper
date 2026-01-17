---
title: 'Health Check'
weight: 1
slug: 'health'
---

# Health Check

The health check will answer with status `200 OK` on every port with the configured
[`health_path` setting](/configuration/block/settings) or the
[`COUPER_HEALTH_PATH` environment variable](/configuration/command-line#observation-options).

As soon as the gateway instance receives a `SIGINT` or `SIGTERM` the check will return a status
`500 Internal Server Error`.

A shutdown delay ([environment variable `COUPER_TIMING_SHUTDOWN_DELAY`](/configuration/command-line#timing-environment-variables))
allows the server to finish all running requests and gives a load balancer time to pick another gateway instance.
After this delay the server goes into shutdown mode and no new requests will be accepted.

The shutdown timings ([`COUPER_TIMING_SHUTDOWN_TIMEOUT`](/configuration/command-line#timing-environment-variables))
default to `0` which means no delaying with development setups.
