package pipeline

import (
	"container/heap"
	"sync"
	"time"
)

// maxJitterMs is the maximum jitter buffer size in milliseconds.
const maxJitterMs = 200

// JitterBuffer stores audio frames for a single stream, reordered by seq.
type JitterBuffer struct {
	mu       sync.Mutex
	heap     frameHeap
	nextSeq  uint32
	init     bool
	lastPush time.Time
	timeout  time.Duration
}

// Frame holds a received audio packet before reordering.
type Frame struct {
	Seq        uint32
	Timestamp  int64
	OpusData   []byte
	Energy     float32
	IsSilence  bool
	SampleRate uint32
	Arrival    time.Time
}

// NewJitterBuffer creates a buffer with 200ms target.
func NewJitterBuffer() *JitterBuffer {
	return &JitterBuffer{
		timeout: maxJitterMs * time.Millisecond,
	}
}

// Push adds a frame to the buffer. Returns frames ready for output in order.
func (b *JitterBuffer) Push(f *Frame) []*Frame {
	b.mu.Lock()
	defer b.mu.Unlock()

	f.Arrival = time.Now()
	b.lastPush = time.Now()

	// Initialize sequence tracking
	if !b.init {
		b.nextSeq = f.Seq
		b.init = true
	}

	// Discard late frames
	if f.Seq < b.nextSeq {
		return nil
	}

	heap.Push(&b.heap, f)

	var ready []*Frame
	for b.heap.Len() > 0 {
		top := b.heap[0]
		if top.Seq == b.nextSeq {
			ready = append(ready, heap.Pop(&b.heap).(*Frame))
			b.nextSeq++
		} else if top.Seq < b.nextSeq {
			heap.Pop(&b.heap) // discard out-of-order
		} else {
			break
		}
	}

	// If buffer has queued frames > maxJitterMs old, force output
	if b.heap.Len() > 0 {
		oldest := b.heap[0]
		if time.Since(oldest.Arrival) > b.timeout {
			ready = append(ready, heap.Pop(&b.heap).(*Frame))
			b.nextSeq = oldest.Seq + 1
		}
	}

	return ready
}

// StallDuration returns how long since the last frame was pushed.
func (b *JitterBuffer) StallDuration() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.lastPush.IsZero() {
		return 0
	}
	return time.Since(b.lastPush)
}

// --- Min-heap by Seq ---

type frameHeap []*Frame

func (h frameHeap) Len() int           { return len(h) }
func (h frameHeap) Less(i, j int) bool { return h[i].Seq < h[j].Seq }
func (h frameHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *frameHeap) Push(x any)        { *h = append(*h, x.(*Frame)) }
func (h *frameHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
