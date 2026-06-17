package audio

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeWSFrame(t *testing.T) {
	original := &AudioFrame{
		Type:       FrameTypeOpus,
		Seq:        42,
		Timestamp:  1718234567890,
		Data:       []byte{0x01, 0x02, 0x03, 0x04},
		SampleRate: 16000,
		Energy:     0.5,
	}

	encoded := original.EncodeWSFrame()

	// Verify header size
	if len(encoded) != 9+len(original.Data) {
		t.Errorf("expected length %d, got %d", 9+len(original.Data), len(encoded))
	}

	// Round-trip
	decoded := DecodeWSFrame(encoded)
	if decoded == nil {
		t.Fatal("DecodeWSFrame returned nil")
	}
	if decoded.Type != FrameTypeOpus {
		t.Errorf("expected type Opus, got %d", decoded.Type)
	}
	if decoded.Seq != 42 {
		t.Errorf("expected seq 42, got %d", decoded.Seq)
	}
	if !bytes.Equal(decoded.Data, original.Data) {
		t.Errorf("data mismatch after round-trip")
	}
}

func TestDecodeWSFrameTooShort(t *testing.T) {
	if f := DecodeWSFrame([]byte{0x00, 0x01}); f != nil {
		t.Errorf("expected nil for short frame")
	}
}

func TestFrameTypeConstants(t *testing.T) {
	if FrameTypeOpus != 0x00 {
		t.Errorf("FrameTypeOpus should be 0x00")
	}
	if FrameTypeSilence != 0x01 {
		t.Errorf("FrameTypeSilence should be 0x01")
	}
}

func TestChannelStreamKey(t *testing.T) {
	key := ChannelStreamKey{ChannelID: "ch_1", StreamID: "user_1"}
	if key.ChannelID != "ch_1" || key.StreamID != "user_1" {
		t.Errorf("unexpected ChannelStreamKey fields")
	}
}
