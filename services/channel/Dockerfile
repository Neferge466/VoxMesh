FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /channel-service ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /channel-service /usr/local/bin/channel-service
EXPOSE 8080
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/channel-service"]
