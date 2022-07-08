# Metrics

- [Metrics](#metrics)
  - [Prometheus](#prometheus)
  - [Grafana Dashboard](#grafana-dashboard)
- [Preview](#preview)
  - [Developers](#developers)

Our metrics feature is [beta](./BETA.md) for now. However, we want to provide some core metrics which can be directly collected from Couper.

## Prometheus

Couper provides a built-in [Prometheus](https://prometheus.io/) exporter. It is configurable with [settings](./REFERENCE.md#settings-block) like port or `service_name` label. If enabled the default scrape target port is `9090`.

## Grafana Dashboard

Couper provides a maintained Grafana dashboard which you can find here: [grafana.json](./../grafana.json)
and import this JSON model to your grafana instance.
You may have to set your Datasource to your Prometheus one.

If you're missing some configuration options or have feedback: Feel free to open a [discussion](https://github.com/avenga/couper/discussions) or
an [issue](https://github.com/avenga/couper/issues) if something does not work as expected or shown values does not make any sense.

### Preview

![dashboard](/img/grafana.png)

## Developers

If you are interested in contributing to our metrics or refine the grafana dashboard: `make docker-telemetry` will spin up the stack for you.
