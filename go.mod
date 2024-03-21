module github.com/coupergateway/couper

go 1.21

require (
	github.com/docker/go-units v0.5.0
	github.com/fatih/color v1.13.0
	github.com/getkin/kin-openapi v0.110.0
	github.com/hashicorp/hcl/v2 v2.20.0
	github.com/jimlambrt/go-oauth-pkce-code-verifier v0.0.0-20201220003123-6363600dffda
	github.com/prometheus/client_golang v1.17.0
	github.com/prometheus/client_model v0.5.0
	github.com/rs/xid v1.4.0
	github.com/russellhaering/gosaml2 v0.9.0
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/sirupsen/logrus v1.9.0
	github.com/zclconf/go-cty v1.14.4
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.44.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
	go.opentelemetry.io/otel/exporters/prometheus v0.44.0
	go.opentelemetry.io/otel/metric v1.21.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.opentelemetry.io/otel/sdk/metric v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
	golang.org/x/crypto v0.17.0
	golang.org/x/net v0.17.0
	google.golang.org/grpc v1.59.0
)

require (
	github.com/algolia/algoliasearch-client-go/v3 v3.26.0
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.0
)

require (
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/beevik/etree v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/invopop/yaml v0.2.0 // indirect
	github.com/jonboulle/clockwork v0.3.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/hashicorp/hcl/v2 v2.20.0 => github.com/johakoch/hcl/v2 v2.0.0-20240321104555-6066cf57fe8f

replace github.com/zclconf/go-cty v1.14.4 => github.com/johakoch/go-cty v0.0.0-20240321100345-e5b2277ac4dc
