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
	docker run --rm -v $(CURDIR):/go/app -w /go/app golang sh -c "go test -short -count 1 -v -timeout 300s -race ./..."

coverage: test-coverage test-coverage-show

test-coverage:
	go test -short -timeout 300s -covermode=count -coverprofile=ac.coverage ./accesscontrol
	go test -short -timeout 300s -covermode=count -coverprofile=cache.coverage ./cache
	go test -short -timeout 300s -covermode=count -coverprofile=command.coverage ./command
	go test -short -timeout 300s -covermode=count -coverprofile=config.coverage ./config
	go test -short -timeout 300s -covermode=count -coverprofile=errors.coverage ./errors
	go test -short -timeout 300s -covermode=count -coverprofile=eval.coverage ./eval
	go test -short -timeout 300s -covermode=count -coverprofile=handler.coverage ./handler
	go test -short -timeout 300s -covermode=count -coverprofile=producer.coverage ./handler/producer
	go test -short -timeout 300s -covermode=count -coverprofile=logging.coverage ./logging
	go test -short -timeout 300s -covermode=count -coverprofile=server.coverage ./server
	go test -short -timeout 300s -covermode=count -coverprofile=main.coverage ./

convert-test-coverage:
	gocov convert ac.coverage > ac.gocov
	gocov convert cache.coverage > cache.gocov
	gocov convert command.coverage > command.gocov
	gocov convert config.coverage > config.gocov
	gocov convert errors.coverage > errors.gocov
	gocov convert eval.coverage > eval.gocov
	gocov convert handler.coverage > handler.gocov
	gocov convert producer.coverage > producer.gocov
	gocov convert logging.coverage > logging.gocov
	gocov convert server.coverage > server.gocov
	gocov convert main.coverage > main.gocov

test-coverage-show:
	go tool cover -html=ac.coverage
	go tool cover -html=cache.coverage
	go tool cover -html=command.coverage
	go tool cover -html=config.coverage
	go tool cover -html=docs.coverage
	go tool cover -html=errors.coverage
	go tool cover -html=eval.coverage
	go tool cover -html=handler.coverage
	go tool cover -html=producer.coverage
	go tool cover -html=logging.coverage
	go tool cover -html=server.coverage
	go tool cover -html=main.coverage

.PHONY: mtls-certificates
mtls-certificates:
	time go run internal/tls/cli/main.go
