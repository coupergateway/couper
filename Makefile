.PHONY: docker-telemetry build generate image
.PHONY: test test-docker coverage test-coverage convert-test-coverage test-coverage-show

GO_VERSION := 1.23

build:
	go build -race -v -o couper main.go

.PHONY: update-modules
update-modules:
	go get -u
	go mod tidy

docker-telemetry:
	docker compose -f telemetry/docker-compose.yaml pull
	docker compose -f telemetry/docker-compose.yaml up --build

generate:
	go generate main.go

generate-docs:
	go run config/generate/main.go

image:
	docker build -t coupergateway/couper:latest .

test:
	go test -v -short -race -count 1 -timeout 300s ./...

test-docker:
	docker run --rm -v $(CURDIR):/go/app -w /go/app golang:$(GO_VERSION) sh -c "go test -short -count 1 -v -timeout 300s -race ./..."

coverage: test-coverage test-coverage-show

test-coverage:
	go test -v -short -vet=off -timeout 300s -coverprofile=c.out ./...

test-coverage-show:
	go tool cover -html=c.out

.PHONY: mtls-certificates
mtls-certificates:
	time go run internal/tls/cli/main.go
