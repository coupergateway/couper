---
title: 'Tracing'
weight: 4
slug: 'tracing'
---

# Tracing

Our tracing feature is [beta](/getting-started/beta-features) for now. Couper supports distributed tracing using [OpenTelemetry](https://opentelemetry.io/) and the [W3C Trace Context](https://www.w3.org/TR/trace-context/) standard.

## Configuration

Enable tracing with `beta_traces = true` in the [settings](/configuration/block/settings) block or the `-beta-traces` CLI flag. Couper exports trace spans via gRPC to an OpenTelemetry collector at `beta_traces_endpoint` (default `localhost:4317`).

The `beta_service_name` setting is used as the service name in exported spans.

## Trace Propagation

Couper supports [W3C Trace Context](https://www.w3.org/TR/trace-context/) propagation via the `traceparent` HTTP header. Two settings control how incoming trace context is handled:

- **`beta_traces_trust_parent`**: When enabled, Couper extracts the `traceparent` header from incoming requests and uses it as the parent span context. This connects Couper's spans to the calling service's trace, enabling end-to-end distributed tracing across services.

- **`beta_traces_parent_only`**: When enabled, Couper only creates trace spans for requests that carry a `traceparent` header. Requests without this header are not traced. This is useful for reducing trace volume in environments where only specific traced requests are of interest.

## Example

```hcl
server {
  api {
    endpoint "/" {
      proxy {
        backend = "my-backend"
      }
    }
  }
}

definitions {
  backend "my-backend" {
    origin = "https://httpbin.org"
  }
}

settings {
  beta_traces         = true
  beta_traces_endpoint = "localhost:4317"
  beta_service_name    = "my-service"

  # Connect to incoming trace context
  beta_traces_trust_parent = true

  # Only trace requests with a traceparent header
  # beta_traces_parent_only = true
}
```

## Developers

If you are interested in contributing to our tracing support: `make docker-telemetry` will spin up the telemetry stack (including Jaeger) for you.
