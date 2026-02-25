---
title: 'Couper Documentation'
---

# Couper Documentation

Couper is a lightweight open-source API gateway that acts as an entry point for clients to your application (frontend API gateway) and an exit point to upstream services (upstream API gateway).

It adds access control, observability, and back-end connectivity on a separate layer. This will keep your core application code more simple.

Couper does not need any special development skills and offers easy configuration and integration.

## Architectural Overview

![architecture](/img/architecture.png)

| Entity           | Description                                                                          |
|:-----------------|:-------------------------------------------------------------------------------------|
| Frontend         | Browser, App or API Client that sends requests to Couper.                            |
| Frontend API     | Couper acts as an entry point for clients to your application.                       |
| Backend Service  | Your core application - no matter which technology, if monolithic or micro-services. |
| Upstream API     | Couper acts as an exit point to upstream services for your application.              |
| Remote Service   | Any upstream service or system which is accessible via HTTP.                         |
| Protected System | Representing a service or system that offers protected resources via HTTP.           |

[Now, let's get started!](/getting-started/running-couper)
