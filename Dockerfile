FROM golang:1.14 AS builder

WORKDIR /go/src/app
COPY . .

ENV GOFLAGS="-mod=vendor"

RUN go generate && \
	CGO_ENABLED=0 go build -v -o /couper main.go && \
	ls -lh /couper

FROM scratch
# copy debian tls ca certs (from golang image)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /couper /couper

COPY public/couper.hcl /conf/
COPY public/index.html /htdocs/
WORKDIR /conf
ENV COUPER_LOG_FORMAT=json
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/couper", "run"]
