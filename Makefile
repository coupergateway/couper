image:
	docker build -t avenga/couper:latest .

test:
	go test -v -short -race -timeout 60s ./...

test-coverage:
	go test -short -timeout 60s -covermode=count -coverprofile=ac.coverage ./accesscontrol
	go test -short -timeout 60s -covermode=count -coverprofile=eval.coverage ./eval
	go test -short -timeout 60s -covermode=count -coverprofile=config.coverage ./config
	go test -short -timeout 60s -covermode=count -coverprofile=handler.coverage ./handler
	go test -short -timeout 60s -covermode=count -coverprofile=server.coverage ./server
	$(MAKE) test-coverage-show

test-coverage-show:
	go tool cover -html=ac.coverage
	go tool cover -html=eval.coverage
	go tool cover -html=config.coverage
	go tool cover -html=handler.coverage
	go tool cover -html=server.coverage
