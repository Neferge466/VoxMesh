package grpcserver

import (
	"context"
	"io"
	slogx "github.com/voxmesh/pkg/log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/voxmesh/audio-mixer/api/audio"
	"github.com/voxmesh/audio-mixer/internal/pipeline"
)

// Server implements the AudioRelay gRPC service.
type Server struct {
	addr string
	gs   *grpc.Server

	mu     sync.RWMutex
	mixers map[string]*pipeline.Mixer // channelID → mixer
	codec  pipeline.Codec
}

// New creates a gRPC server for audio relay.
func New(addr string, codec pipeline.Codec) *Server {
	return &Server{
		addr:   addr,
		codec:  codec,
		mixers: make(map[string]*pipeline.Mixer),
	}
}

// Start begins listening on the configured address.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.gs = grpc.NewServer(
		grpc.MaxConcurrentStreams(1000),
	)
	audio.RegisterAudioRelayServer(s.gs, s)

	slogx.Info("[audio-mixer] gRPC listening on %s", s.addr)
	return s.gs.Serve(lis)
}

// GracefulStop shuts down the gRPC server, waiting for active RPCs to finish.
func (s *Server) GracefulStop() {
	if s.gs != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		done := make(chan struct{})
		go func() {
			s.gs.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			s.gs.Stop() // force after timeout
		}
	}
}

// RelayAudio handles a bidirectional audio stream from a WS Gateway client.
func (s *Server) RelayAudio(stream audio.AudioRelay_RelayAudioServer) error {
	// Track which channels this stream is associated with
	userChannels := make(map[string]string) // userID → channelID

	defer func() {
		// Cleanup: remove user from channel mixers
		for uid, chID := range userChannels {
			s.mu.RLock()
			m := s.mixers[chID]
			s.mu.RUnlock()
			if m != nil {
				m.RemoveUser(uid)
			}
		}
	}()

	for {
		pkt, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Get or create channel mixer
		s.mu.RLock()
		mixer, ok := s.mixers[pkt.ChannelId]
		s.mu.RUnlock()

		if !ok {
			s.mu.Lock()
			mixer, ok = s.mixers[pkt.ChannelId]
			if !ok {
				mixer = pipeline.NewMixer(pkt.ChannelId, s.codec,
					func(out *audio.MixedAudioPacket) {
						// Route output back to the WS Gateway via gRPC stream
						if err := stream.Send(out); err != nil {
							slogx.Info("[audio-mixer] send error: %v", err)
						}
					},
				)
				s.mixers[pkt.ChannelId] = mixer
			}
			s.mu.Unlock()
		}

		userChannels[pkt.UserId] = pkt.ChannelId
		mixer.IngestFrame(pkt)
	}
}

// Ensure interface compliance
var _ audio.AudioRelayServer = (*Server)(nil)
