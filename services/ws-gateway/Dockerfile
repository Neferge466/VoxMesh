FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /ws-gateway ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /ws-gateway /usr/local/bin/ws-gateway
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/ws-gateway"]
