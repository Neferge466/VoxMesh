FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/auth/go.mod services/auth/go.sum ./
COPY services/pkg/ /pkg/
COPY services/auth/ .
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 go build -o /auth-service ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /auth-service /usr/local/bin/auth-service
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/auth-service"]
