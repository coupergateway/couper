FROM golang:1.14 AS builder

WORKDIR /go/src/app
COPY . .

ENV GOFLAGS="-mod=vendor"

RUN go generate && \
	CGO_ENABLED=0 go build -v -o /couper main.go && \
	ls -lh /couper

RUN mkdir /conf

FROM scratch
# copy ssl certs
COPY --from=builder /usr/share/ca-certificates/ /usr/share/ca-certificates/
COPY --from=builder /etc/ssl /etc/ssl
COPY --from=builder /couper /couper
COPY --from=builder /conf /conf
EXPOSE 8080
WORKDIR /conf
USER 1000:1000
ENTRYPOINT ["/couper"]
