package telemetry

import "go.opentelemetry.io/otel/attribute"

const keyPrefix = "couper."

var KeyBackend = attribute.Key(keyPrefix + "backend")
var KeyEndpoint = attribute.Key(keyPrefix + "endpoint")
var KeyOrigin = attribute.Key(keyPrefix + "origin")
var KeyUID = attribute.Key(keyPrefix + "uid")
