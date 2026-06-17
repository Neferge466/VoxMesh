//go:build !cgo

package codec

// NewCodec returns a noop pass-through codec for local development.
// When CGO_ENABLED=1, factory_cgo.go is used instead.
func NewCodec() *NoopCodec {
	return NewNoopCodec()
}
