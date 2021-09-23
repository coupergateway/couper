package telemetry

import "go.opentelemetry.io/otel/attribute"

const keyPrefix = "couper."

var KeyEndpoint = attribute.Key(keyPrefix + "endpoint")
var KeyOrigin = attribute.Key(keyPrefix + "origin")
