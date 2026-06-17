// Package audio defines audio frame structures shared between services.
package audio

import "encoding/binary"

// FrameType identifies the audio frame variant.
type FrameType uint8

const (
	FrameTypeOpus    FrameType = 0x00
	FrameTypeSilence FrameType = 0x01
)

// AudioFrame is a decoded audio frame in transit between services.
type AudioFrame struct {
	Type       FrameType
	UserID     string
	DeviceID   string
	ChannelID  string
	Seq        uint32
	Timestamp  int64 // Unix milliseconds
	SampleRate uint16
	Energy     float32
	Data       []byte // Opus-encoded audio (or empty for silence)
}

// WebSocket binary frame layout (network byte order):
// Byte 0:      FrameType (0x00 = Opus, 0x01 = Silence)
// Bytes 1-4:   Seq (uint32, big-endian)
// Bytes 5-8:   Timestamp ms (uint32, big-endian)
// Bytes 9-N:   Opus data
const wsFrameHeaderSize = 9

// EncodeWSFrame serializes an AudioFrame for WebSocket binary transport.
func (f *AudioFrame) EncodeWSFrame() []byte {
	buf := make([]byte, wsFrameHeaderSize+len(f.Data))
	buf[0] = byte(f.Type)
	binary.BigEndian.PutUint32(buf[1:5], f.Seq)
	binary.BigEndian.PutUint32(buf[5:9], uint32(f.Timestamp))
	copy(buf[9:], f.Data)
	return buf
}

// DecodeWSFrame parses a WebSocket binary frame into an AudioFrame.
func DecodeWSFrame(data []byte) *AudioFrame {
	if len(data) < wsFrameHeaderSize {
		return nil
	}
	return &AudioFrame{
		Type:      FrameType(data[0]),
		Seq:       binary.BigEndian.Uint32(data[1:5]),
		Timestamp: int64(binary.BigEndian.Uint32(data[5:9])),
		Data:      data[9:],
	}
}

// MixedAudioFrame is a mixed audio frame sent to a specific listener
// (N-1 mix that excludes the listener's own audio).
type MixedAudioFrame struct {
	ChannelID      string
	Seq            uint32
	Timestamp      int64
	SpeakerCount   uint8
	ActiveSpeakers []string
	Data           []byte // Mixed Opus-encoded audio
}

// Channel identity for audio routing.
type ChannelStreamKey struct {
	ChannelID string
	StreamID  string // user_id or device_id
}
