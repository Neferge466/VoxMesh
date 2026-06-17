//go:build cgo

package codec

// NewCodec returns a libopus-backed codec for production use.
// Only compiled when CGO_ENABLED=1 (cgo build tag is auto-set by go when
// any file in the package uses import "C").
func NewCodec() *OpusCodec {
	return NewOpusCodec(48000, 1)
}
