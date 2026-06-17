FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/gateway-coordinator/go.mod services/gateway-coordinator/go.sum ./
COPY services/pkg/ /pkg/
RUN go mod download
COPY services/gateway-coordinator/ .
RUN CGO_ENABLED=0 go build -o /gateway-coordinator ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /gateway-coordinator /usr/local/bin/gateway-coordinator
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/gateway-coordinator"]
