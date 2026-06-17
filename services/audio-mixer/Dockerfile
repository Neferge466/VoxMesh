FROM golang:1.24-alpine AS builder
RUN apk add --no-cache opus-dev pkgconfig gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /audio-mixer ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates opus
COPY --from=builder /audio-mixer /usr/local/bin/audio-mixer
EXPOSE 9000
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/audio-mixer"]
