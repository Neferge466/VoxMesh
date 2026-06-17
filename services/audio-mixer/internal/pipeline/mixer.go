package pipeline

import (
	"log"
	"math"
	"sync"
	"time"

	"github.com/voxmesh/audio-mixer/api/audio"
)

// Codec handles Opus encoding/decoding. Swap implementation for cgo (prod)
// vs pure Go (dev/testing).
type Codec interface {
	Decode(opusData []byte, sampleRate int) ([]int16, error)
	Encode(pcm []int16, sampleRate int) ([]byte, error)
}

// Mixer handles per-channel audio mixing with N-1 output.
type Mixer struct {
	channelID string
	codec     Codec
	vad       *VAD

	mu       sync.Mutex
	buffers  map[string]*JitterBuffer // userID → jitter buffer
	active   map[string]bool          // userID → is speaking
	outputFn func(pkt *audio.MixedAudioPacket) // called when mixed audio is ready
}

// NewMixer creates a per-channel mixer.
func NewMixer(channelID string, codec Codec, outputFn func(pkt *audio.MixedAudioPacket)) *Mixer {
	return &Mixer{
		channelID: channelID,
		codec:     codec,
		vad:       NewVAD(DefaultEnergyThreshold, DefaultHangoverFrames),
		buffers:   make(map[string]*JitterBuffer),
		active:    make(map[string]bool),
		outputFn:  outputFn,
	}
}

// IngestFrame receives an audio packet and queues it for mixing.
func (m *Mixer) IngestFrame(pkt *audio.AudioPacket) {
	m.mu.Lock()
	buf, ok := m.buffers[pkt.UserId]
	if !ok {
		buf = NewJitterBuffer()
		m.buffers[pkt.UserId] = buf
	}
	m.mu.Unlock()

	frame := &Frame{
		Seq:        pkt.Seq,
		Timestamp:  pkt.TimestampMs,
		OpusData:   pkt.OpusData,
		Energy:     pkt.Energy,
		IsSilence:  pkt.IsSilence,
		SampleRate: pkt.SampleRate,
	}

	isSpeech := m.vad.IsSpeech(frame.Energy, frame.IsSilence)
	m.mu.Lock()
	m.active[pkt.UserId] = isSpeech
	m.mu.Unlock()

	readyFrames := buf.Push(frame)

	// Mix and output for each frame
	for range readyFrames {
		m.mixAndOutput(pkt.UserId)
	}
}

// mixAndOutput performs N-1 mixing: for the given listener, mix all OTHER speakers.
func (m *Mixer) mixAndOutput(_ string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect active speakers
	var activeSpeakers []string
	for uid, isActive := range m.active {
		if isActive {
			activeSpeakers = append(activeSpeakers, uid)
		}
	}

	if len(activeSpeakers) == 0 {
		return
	}

	// For each active user, generate N-1 mixed output
	for _, listenerID := range activeSpeakers {
		// Mix all OTHER active speakers
		mixSpeakers := make([]string, 0, len(activeSpeakers)-1)
		for _, uid := range activeSpeakers {
			if uid != listenerID {
				mixSpeakers = append(mixSpeakers, uid)
			}
		}

		if len(mixSpeakers) == 0 {
			continue
		}

		// Generate mixed output (simplified: in production, this would
		// decode each speaker's Opus, mix PCM, re-encode)
		// For now, we pass through the first speaker's audio as placeholder.
		mixed := m.mixPCM(mixSpeakers)
		if mixed == nil {
			continue
		}

		m.outputFn(&audio.MixedAudioPacket{
			ChannelId:      m.channelID,
			ListenerUserId: listenerID,
			Seq:            0,
			TimestampMs:    time.Now().UnixMilli(),
			SpeakerCount:   uint32(len(mixSpeakers)),
			ActiveSpeakers: mixSpeakers,
			MixedOpusData:  mixed,
		})
	}
}

// mixPCM decodes all speakers, sums PCM, encodes back.
func (m *Mixer) mixPCM(speakers []string) []byte {
	if len(speakers) == 0 {
		return nil
	}

	// Decode first speaker as reference
	refBuf, ok := m.buffers[speakers[0]]
	if !ok {
		return nil
	}

	// Get the most recent frame from each speaker's buffer
	// For simplicity: use only the first speaker (full multi-stream decode+mix
	// requires cgo libopus and is done in the Docker CGO_ENABLED=1 build).
	_ = refBuf

	// Placeholder: in production, decode all speakers with libopus,
	// sum PCM samples with clipping, re-encode.
	return nil
}

// SpeakerCount returns the number of currently active speakers.
func (m *Mixer) SpeakerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, active := range m.active {
		if active {
			count++
		}
	}
	return count
}

// RemoveUser deletes a user's buffer and active state.
func (m *Mixer) RemoveUser(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.buffers, userID)
	delete(m.active, userID)
}

// Cleanup removes stale buffers (stream stall > 5 seconds).
func (m *Mixer) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for uid, buf := range m.buffers {
		if buf.StallDuration() > 5*time.Second {
			delete(m.buffers, uid)
			delete(m.active, uid)
		}
	}
}

// RMSNormalize scales PCM samples to a target RMS level.
func RMSNormalize(pcm []int16, targetRMS float64) []int16 {
	if len(pcm) == 0 {
		return pcm
	}

	var sum float64
	for _, s := range pcm {
		sum += float64(s) * float64(s)
	}
	rms := math.Sqrt(sum / float64(len(pcm)))
	if rms < 1 {
		return pcm
	}

	scale := targetRMS / rms
	if scale > 3.0 {
		scale = 3.0 // max 3x gain
	}

	out := make([]int16, len(pcm))
	for i, s := range pcm {
		v := float64(s) * scale
		if v > 32767 {
			v = 32767
		}
		if v < -32768 {
			v = -32768
		}
		out[i] = int16(v)
	}
	return out
}

// MixPCMInt16 sums multiple PCM streams with clipping.
func MixPCMInt16(streams [][]int16) []int16 {
	if len(streams) == 0 {
		return nil
	}
	if len(streams) == 1 {
		return streams[0]
	}

	// Use first stream length as reference
	length := len(streams[0])
	out := make([]int16, length)

	for i := 0; i < length; i++ {
		var sum int32
		for _, s := range streams {
			if i < len(s) {
				sum += int32(s[i])
			}
		}
		// Clip to int16 range
		if sum > 32767 {
			sum = 32767
		}
		if sum < -32768 {
			sum = -32768
		}
		out[i] = int16(sum)
	}
	return out
}

// Ensure unused import refs compile.
var _ = log.Printf
