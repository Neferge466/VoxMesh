package pipeline

// VAD (Voice Activity Detection) using energy threshold.
type VAD struct {
	threshold   float32
	hangover    int     // frames to keep active after energy drops
	counter     int
}

// NewVAD creates a voice activity detector.
// energyThreshold: frames below this energy are silence.
// hangoverFrames: number of frames to remain active after energy drops below threshold.
func NewVAD(energyThreshold float32, hangoverFrames int) *VAD {
	return &VAD{
		threshold: energyThreshold,
		hangover:  hangoverFrames,
	}
}

// IsSpeech returns true if the frame is likely speech.
func (v *VAD) IsSpeech(energy float32, isSilence bool) bool {
	if isSilence {
		if v.counter > 0 {
			v.counter--
			return v.counter > 0
		}
		return false
	}

	if energy > v.threshold {
		v.counter = v.hangover
		return true
	}

	if v.counter > 0 {
		v.counter--
		return true
	}
	return false
}

// Default speech energy threshold for Opus frames.
const DefaultEnergyThreshold = 0.005
const DefaultHangoverFrames = 10 // ~200ms at 20ms frames
