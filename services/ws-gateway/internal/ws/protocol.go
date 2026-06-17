package ws

import "encoding/json"

// Envelope wraps all WebSocket JSON messages.
type Envelope struct {
	Type        string          `json:"type"`
	ID          string          `json:"id"`
	TimestampMs int64           `json:"timestamp_ms"`
	Payload     json.RawMessage `json:"payload"`
}

// Client-to-server payloads.

type JoinChannelPayload struct {
	ChannelID string `json:"channel_id"`
	Password  string `json:"password,omitempty"`
}

type LeaveChannelPayload struct {
	ChannelID string `json:"channel_id"`
}

// Server-to-client payloads.

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ChannelJoinedPayload struct {
	ChannelID string   `json:"channel_id"`
	Members   []Member `json:"members"`
}

type Member struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Speaking    bool   `json:"speaking"`
	Muted       bool   `json:"muted"`
	ClientType  string `json:"client_type"`
}

type PresenceUpdatePayload struct {
	ChannelID string `json:"channel_id"`
	Members   []Member `json:"members"`
}

type UserSpeakingPayload struct {
	UserID   string `json:"user_id"`
	Speaking bool   `json:"speaking"`
}

type ChatMessagePayload struct {
	SenderID      string `json:"sender_id"`
	SenderName    string `json:"sender_name"`
	ChannelID     string `json:"channel_id"`
	Content       string `json:"content"`
}

// WebRTC signaling payloads.

type SDPSignalPayload struct {
	SDP      string `json:"sdp,omitempty"`
	SenderID string `json:"sender_id"`
}

type ICECandidatePayload struct {
	Candidate string `json:"candidate"`
	SenderID  string `json:"sender_id"`
}

// Message type constants.
const (
	TypeAuthenticate    = "authenticate"
	TypeAuthenticated   = "authenticated"
	TypeError           = "error"
	TypeJoinChannel     = "join_channel"
	TypeLeaveChannel    = "leave_channel"
	TypeChannelJoined   = "channel_joined"
	TypeChannelLeft     = "channel_left"
	TypePresenceUpdate  = "presence_update"
	TypeUserSpeaking    = "user_speaking"
	TypeSetMute         = "set_mute"
	TypeSetDeafen       = "set_deafen"
	TypeStartSpeaking   = "start_speaking"
	TypeStopSpeaking    = "stop_speaking"
	TypePing            = "ping"
	TypePong            = "pong"
	TypeChatMessage     = "chat_message"
	TypeChannelList     = "channel_list"
	TypeSDPOffer        = "sdp_offer"
	TypeSDPAnswer       = "sdp_answer"
	TypeICECandidate    = "ice_candidate"
)

// NewEnvelope creates a new message envelope.
func NewEnvelope(msgType, id string, payload interface{}) Envelope {
	data, _ := json.Marshal(payload)
	return Envelope{
		Type:    msgType,
		ID:      id,
		Payload: data,
	}
}

// NewError creates an error envelope.
func NewError(id string, code int, message string) Envelope {
	return Envelope{
		Type: TypeError,
		ID:   id,
		Payload: mustMarshal(ErrorPayload{Code: code, Message: message}),
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
