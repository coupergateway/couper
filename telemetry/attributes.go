package telemetry

import "go.opentelemetry.io/otel/attribute"

var KeyEndpoint = attribute.Key("couper.endpoint")
var KeyBackend = attribute.Key("couper.backend")
var KeyOrigin = attribute.Key("couper.origin")
