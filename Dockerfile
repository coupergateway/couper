FROM golang:1.14 AS builder

WORKDIR /go/src/app
COPY . .

ENV GOFLAGS="-mod=vendor"

RUN go generate && \
	CGO_ENABLED=0 go build -v -o /couper main.go && \
	ls -lh /couper

RUN mkdir /conf

FROM scratch
COPY --from=builder /couper /couper
COPY --from=builder /conf /conf
EXPOSE 8080
WORKDIR /conf
ENTRYPOINT ["/couper"]
