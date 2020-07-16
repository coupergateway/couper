FROM golang:1.14

WORKDIR /go/src/app
COPY . .

RUN go test -v -timeout=30s -race ./...

RUN go build main.go
