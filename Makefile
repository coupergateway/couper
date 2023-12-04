.PHONY: docker-telemetry build generate image
.PHONY: test test-docker coverage test-coverage convert-test-coverage test-coverage-show

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
	docker run --rm -v $(CURDIR):/go/app -w /go/app golang:1.20 sh -c "go test -short -count 1 -v -timeout 300s -race ./..."

coverage: test-coverage test-coverage-show

test-coverage:
	go test -short -timeout 300s -covermode=count -coverprofile=ac.out ./accesscontrol
	go test -short -timeout 300s -covermode=count -coverprofile=cache.out ./cache
	go test -short -timeout 300s -covermode=count -coverprofile=command.out ./command
	go test -short -timeout 300s -covermode=count -coverprofile=config.out ./config
	go test -short -timeout 300s -covermode=count -coverprofile=errors.out ./errors
	go test -short -timeout 300s -covermode=count -coverprofile=eval.out ./eval
	go test -short -timeout 300s -covermode=count -coverprofile=handler.out ./handler
	go test -short -timeout 300s -covermode=count -coverprofile=producer.out ./handler/producer
	go test -short -timeout 300s -covermode=count -coverprofile=logging.out ./logging
	go test -short -timeout 300s -covermode=count -coverprofile=server.out ./server
	go test -short -timeout 300s -covermode=count -coverprofile=main.out ./

test-coverage-show:
	go tool cover -html=ac.out
	go tool cover -html=cache.out
	go tool cover -html=command.out
	go tool cover -html=config.out
	go tool cover -html=docs.out
	go tool cover -html=errors.out
	go tool cover -html=eval.out
	go tool cover -html=handler.out
	go tool cover -html=producer.out
	go tool cover -html=logging.out
	go tool cover -html=server.out
	go tool cover -html=main.out

.PHONY: mtls-certificates
mtls-certificates:
	time go run internal/tls/cli/main.go
