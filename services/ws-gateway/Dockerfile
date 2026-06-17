FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/ws-gateway/go.mod services/ws-gateway/go.sum ./
RUN go mod download
COPY services/ws-gateway/ .
RUN CGO_ENABLED=0 go build -o /ws-gateway ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /ws-gateway /usr/local/bin/ws-gateway
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/ws-gateway"]
