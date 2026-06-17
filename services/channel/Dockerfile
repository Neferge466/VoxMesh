FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/channel/go.mod services/channel/go.sum ./
RUN go mod download
COPY services/channel/ .
RUN CGO_ENABLED=0 go build -o /channel-service ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /channel-service /usr/local/bin/channel-service
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/channel-service"]
