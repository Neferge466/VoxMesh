FROM golang:1.26-alpine AS builder
RUN apk add --no-cache opus-dev pkgconfig gcc musl-dev
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY services/audio-mixer/go.mod services/audio-mixer/go.sum ./
COPY services/pkg/ /pkg/
RUN go mod tidy
RUN go mod download
COPY services/audio-mixer/ .
RUN CGO_ENABLED=1 go build -o /audio-mixer ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates opus
COPY --from=builder /audio-mixer /usr/local/bin/audio-mixer
EXPOSE 9000
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/audio-mixer"]
