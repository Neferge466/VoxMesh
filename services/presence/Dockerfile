FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/presence/go.mod services/presence/go.sum ./
COPY services/pkg/ /pkg/
RUN go mod download
COPY services/presence/ .
RUN CGO_ENABLED=0 go build -o /presence-manager ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /presence-manager /usr/local/bin/presence-manager
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/presence-manager"]
