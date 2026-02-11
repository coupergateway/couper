.PHONY: docker-telemetry build generate generate-llmstxt image
.PHONY: test test-docker coverage test-coverage convert-test-coverage test-coverage-show

GO_VERSION := 1.25

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

generate-llmstxt:
	go run -tags exclude config/generate/llmstxt/main.go

generate-docs: generate-llmstxt
	go run config/generate/main.go

.PHONY: serve-docs
serve-docs: generate-docs
	cd docs/website && hugo server

image:
	docker build -t coupergateway/couper:latest .

test:
	@echo "Running tests..."
	@for PACKAGE in $$(go list ./...); do \
		echo "Testing $$PACKAGE"; \
		if go test -v -timeout 90s -race -count=1 $$PACKAGE; then \
			echo "\033[32m✔ $$PACKAGE PASSED\033[0m"; \
		else \
			echo "\033[31m✖ $$PACKAGE FAILED\033[0m"; \
			exit 1; \
		fi; \
	done
	@echo "\033[32m✔ All tests passed!\033[0m"



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
