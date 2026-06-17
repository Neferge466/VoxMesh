FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/notification/go.mod services/notification/go.sum ./
COPY services/pkg/ /pkg/
COPY services/notification/ .
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 go build -o /notification-service ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /notification-service /usr/local/bin/notification-service
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/notification-service"]
