
test:
	go test -v -short -race -timeout 30s ./...

test-coverage:
	go test -short -timeout 30s -covermode=count -coverprofile=config.coverage ./config
	go test -short -timeout 30s -covermode=count -coverprofile=handler.coverage ./handler
	go test -short -timeout 30s -covermode=count -coverprofile=server.coverage ./server
	$(MAKE) test-coverage-show

test-coverage-show:
	go tool cover -html=config.coverage
	go tool cover -html=handler.coverage
	go tool cover -html=server.coverage
