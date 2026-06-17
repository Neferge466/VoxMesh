FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway-coordinator ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /gateway-coordinator /usr/local/bin/gateway-coordinator
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/gateway-coordinator"]
