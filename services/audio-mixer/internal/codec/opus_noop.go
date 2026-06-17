//go:build !cgo

package codec

// NoopCodec is the development-only Opus codec.
//
// It returns empty slices for both Decode and Encode — no real audio processing.
// This is intentional:
//
// 1. The Audio Mixer's only consumer is the embedded device bridge path
//    (ESP32 → MQTT → Mixer → gRPC → Web). This path is NOT exercised
//    during local frontend development — web clients use WebRTC P2P SRTP
//    for audio, which bypasses the mixer entirely.
//
// 2. Real Opus processing requires CGO + libopus-dev. Requiring every
//    developer to install these C dependencies slows down onboarding
//    for no benefit (they can't test the embedded path locally anyway).
//
// 3. In production Docker builds (CGO_ENABLED=1), this file is excluded
//    by the !cgo build constraint, and opus_cgo.go is used instead.
//
// To test the embedded audio path locally, install libopus-dev and build
// with CGO_ENABLED=1.
type NoopCodec struct{}

func NewNoopCodec() *NoopCodec {
	return &NoopCodec{}
}

func (n *NoopCodec) Decode(opusData []byte, _ int) ([]int16, error) {
	return make([]int16, 0), nil
}

func (n *NoopCodec) Encode(pcm []int16, _ int) ([]byte, error) {
	return nil, nil
}
