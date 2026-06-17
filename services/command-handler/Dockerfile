FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /command-handler ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /command-handler /usr/local/bin/command-handler
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/command-handler"]
