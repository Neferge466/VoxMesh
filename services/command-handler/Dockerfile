FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/command-handler/go.mod services/command-handler/go.sum ./
COPY services/pkg/ /pkg/
COPY services/command-handler/ .
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 go build -o /command-handler ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /command-handler /usr/local/bin/command-handler
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/command-handler"]
