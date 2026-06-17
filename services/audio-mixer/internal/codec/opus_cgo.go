// Package codec provides Opus encode/decode via cgo libopus.
// This file is only compiled when CGO_ENABLED=1 (import "C" auto-sets the cgo build tag).
package codec

/*
#cgo LDFLAGS: -lopus
#include <opus/opus.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// OpusCodec implements the pipeline Codec interface using libopus via cgo.
type OpusCodec struct {
	sampleRate int
	channels   int
}

// NewOpusCodec creates a real libopus-backed codec.
func NewOpusCodec(sampleRate, channels int) *OpusCodec {
	return &OpusCodec{sampleRate: sampleRate, channels: channels}
}

func (c *OpusCodec) Decode(opusData []byte, sampleRate int) ([]int16, error) {
	if len(opusData) == 0 {
		return nil, nil
	}

	frameSamples := sampleRate * 20 / 1000 // 20ms Opus frame

	var decErr C.int
	dec := C.opus_decoder_create(C.int(sampleRate), C.int(c.channels), &decErr)
	if decErr != C.OPUS_OK {
		return nil, fmt.Errorf("opus decoder create: %d", int(decErr))
	}
	defer C.opus_decoder_destroy(dec)

	// Max decoded PCM for a 120ms worst-case Opus packet
	maxOut := frameSamples * c.channels * 6
	pcm := make([]int16, maxOut)

	n := C.opus_decode(
		dec,
		(*C.uchar)(unsafe.Pointer(&opusData[0])),
		C.opus_int32(len(opusData)),
		(*C.opus_int16)(unsafe.Pointer(&pcm[0])),
		C.int(frameSamples),
		0, // no FEC
	)
	if n < 0 {
		return nil, fmt.Errorf("opus decode error: %d", int(n))
	}

	total := int(n) * c.channels
	if total > len(pcm) {
		total = len(pcm)
	}
	return pcm[:total], nil
}

func (c *OpusCodec) Encode(pcm []int16, sampleRate int) ([]byte, error) {
	if len(pcm) == 0 {
		return nil, nil
	}

	frameSamples := sampleRate * 20 / 1000 // 20ms per Opus frame

	var encErr C.int
	enc := C.opus_encoder_create(C.int(sampleRate), C.int(c.channels), C.OPUS_APPLICATION_VOIP, &encErr)
	if encErr != C.OPUS_OK {
		return nil, fmt.Errorf("opus encoder create: %d", int(encErr))
	}
	defer C.opus_encoder_destroy(enc)

	// 32kbps — good voice quality at low bandwidth
	C.opus_encoder_ctl(enc, C.OPUS_SET_BITRATE(C.opus_int32(32000)))
	C.opus_encoder_ctl(enc, C.OPUS_SET_COMPLEXITY(C.opus_int32(5)))
	C.opus_encoder_ctl(enc, C.OPUS_SET_SIGNAL(C.OPUS_SIGNAL_VOICE))

	// Allocate output buffer: max 4000 bytes per Opus frame
	maxEncoded := frameSamples * 4
	out := make([]byte, maxEncoded)
	outOff := 0

	// Encode in 20ms chunks; pad last chunk with silence
	chunk := make([]int16, frameSamples)
	for offset := 0; offset < len(pcm); offset += frameSamples {
		end := offset + frameSamples
		if end > len(pcm) {
			end = len(pcm)
		}
		// Zero the chunk (silence padding for incomplete final frame)
		for i := range chunk {
			chunk[i] = 0
		}
		copy(chunk, pcm[offset:end])

		n := C.opus_encode(
			enc,
			(*C.opus_int16)(unsafe.Pointer(&chunk[0])),
			C.int(frameSamples),
			(*C.uchar)(unsafe.Pointer(&out[outOff])),
			C.opus_int32(len(out)-outOff),
		)
		if n < 0 {
			return nil, fmt.Errorf("opus encode error: %d", int(n))
		}
		outOff += int(n)
	}

	return out[:outOff], nil
}
