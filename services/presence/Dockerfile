FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /presence-manager ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /presence-manager /usr/local/bin/presence-manager
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/presence-manager"]
