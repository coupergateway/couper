module github.com/avenga/couper

go 1.19

require (
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/docker/go-units v0.4.0
	github.com/fatih/color v1.13.0
	github.com/getkin/kin-openapi v0.92.0
	github.com/hashicorp/hcl/v2 v2.12.0
	github.com/jimlambrt/go-oauth-pkce-code-verifier v0.0.0-20201220003123-6363600dffda
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rs/xid v1.3.0
	github.com/russellhaering/gosaml2 v0.7.0
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/zclconf/go-cty v1.10.1-0.20220623211957-834994b84991
	go.opentelemetry.io/otel v1.4.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.4.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.4.0
	go.opentelemetry.io/otel/exporters/prometheus v0.27.0
	go.opentelemetry.io/otel/metric v0.27.0
	go.opentelemetry.io/otel/sdk v1.4.0
	go.opentelemetry.io/otel/sdk/metric v0.27.0
	go.opentelemetry.io/otel/trace v1.4.0
	golang.org/x/crypto v0.0.0-20211209193657-4570a0811e8b
	golang.org/x/net v0.0.0-20211209124913-491a49abca63
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.44.0
)

require (
	github.com/algolia/algoliasearch-client-go/v3 v3.26.0
	github.com/golang-jwt/jwt/v4 v4.4.2
	github.com/google/go-cmp v0.5.7
)

require (
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/beevik/etree v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.4.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.27.0 // indirect
	go.opentelemetry.io/proto/otlp v0.12.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/hashicorp/hcl/v2 v2.12.0 => github.com/avenga/hcl v1.0.1-0.20220614110422-d3e10849a937

replace github.com/zclconf/go-cty => github.com/avenga/go-cty v0.0.0-20220627115852-3ddf0a688aea
