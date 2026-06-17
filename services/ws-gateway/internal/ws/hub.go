package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fasthttp/websocket"
	slogx "github.com/voxmesh/pkg/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 65536
)

// Client represents a connected WebSocket client.
type Client struct {
	ConnID      string // unique per connection (supports same user on multiple devices)
	UserID      string
	DisplayName string
	ChannelID   string
	conn        *websocket.Conn
	hub         *Hub
	send        chan []byte
	done        chan struct{}
	mu          sync.Mutex
	Muted       bool
	Deafened    bool
}

func (c *Client) Done() <-chan struct{} { return c.done }

func NewClient(userID, displayName string, conn *websocket.Conn, hub *Hub) *Client {
	cid := hub.connSeq.Add(1)
	return &Client{
		ConnID:      fmt.Sprintf("%s-%d", userID, cid),
		UserID:      userID,
		DisplayName: displayName,
		conn:        conn,
		hub:         hub,
		send:        make(chan []byte, 256),
		done:        make(chan struct{}),
	}
}

func (c *Client) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
	default:
		return nil // drop if buffer full
	}
	return nil
}

func (c *Client) SendBinary(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *Client) ReadPump() {
	defer func() {
		if r := recover(); r != nil {
			slogx.Info("[ws] read pump panic: %v", r)
		}
		close(c.done)
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		switch msgType {
		case websocket.TextMessage:
			c.hub.handleJSON(c, data)
		case websocket.BinaryMessage:
			c.hub.handleBinary(c, data)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if r := recover(); r != nil {
			slogx.Info("[ws] write pump panic: %v", r)
		}
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.mu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		case <-ticker.C:
			c.mu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		}
	}
}

// Hub manages connected clients and message routing.
// Clients are keyed by a unique ConnID (not UserID) so the same user
// can connect from multiple devices simultaneously.
type Hub struct {
	clients      map[string]*Client  // ConnID → Client
	userConns    map[string][]string // UserID → []ConnID
	connSeq      atomic.Uint64
	Register     chan *Client
	Unregister   chan *Client
	mu           sync.RWMutex
	onDisconnect func(userID string)
}

func NewHub(onDisconnect func(userID string)) *Hub {
	return &Hub{
		clients:      make(map[string]*Client),
		userConns:    make(map[string][]string),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		onDisconnect: onDisconnect,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.ConnID] = client
			h.userConns[client.UserID] = append(h.userConns[client.UserID], client.ConnID)
			h.mu.Unlock()
			slogx.Info("[ws] client connected: %s conn=%s", client.UserID, client.ConnID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ConnID]; ok {
				delete(h.clients, client.ConnID)
				close(client.send)
				// Remove this ConnID from userConns
				ids := h.userConns[client.UserID]
				for i, cid := range ids {
					if cid == client.ConnID {
						h.userConns[client.UserID] = append(ids[:i], ids[i+1:]...)
						break
					}
				}
				if len(h.userConns[client.UserID]) == 0 {
					delete(h.userConns, client.UserID)
				}
			}
			h.mu.Unlock()
			if h.onDisconnect != nil {
				h.onDisconnect(client.UserID)
			}
			slogx.Info("[ws] client disconnected: %s conn=%s", client.UserID, client.ConnID)
		}
	}
}

func (h *Hub) GetClient(userID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, cid := range h.userConns[userID] {
		if c, ok := h.clients[cid]; ok {
			return c
		}
	}
	return nil
}

// GetClientsByUser returns all connections for a given user (multi-device).
func (h *Hub) GetClientsByUser(userID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := h.userConns[userID]
	result := make([]*Client, 0, len(ids))
	for _, cid := range ids {
		if c, ok := h.clients[cid]; ok {
			result = append(result, c)
		}
	}
	return result
}

func (h *Hub) BroadcastToChannel(channelID string, msg interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c.ChannelID == channelID {
			c.SendJSON(msg)
		}
	}
}

func (h *Hub) BroadcastToChannelExcept(channelID, excludeUserID string, msg interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c.ChannelID == channelID && c.UserID != excludeUserID {
			c.SendJSON(msg)
		}
	}
}

func (h *Hub) broadcastPresence(channelID string) {
	var members []Member
	h.mu.RLock()
	for _, c := range h.clients {
		if c.ChannelID == channelID {
			members = append(members, Member{
				UserID:      c.UserID,
				DisplayName: c.DisplayName,
				Speaking:    false,
				Muted:       c.Muted,
				ClientType:  "web",
			})
		}
	}
	h.mu.RUnlock()
	h.BroadcastToChannel(channelID,
		NewEnvelope(TypePresenceUpdate, "", PresenceUpdatePayload{
			ChannelID: channelID,
			Members:   members,
		}))
}

func (h *Hub) handleJSON(client *Client, data []byte) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		client.SendJSON(NewError("", 40000, "invalid message format"))
		return
	}

	switch env.Type {
	case TypeJoinChannel:
		var p JoinChannelPayload
		json.Unmarshal(env.Payload, &p)
		oldCh := client.ChannelID
		client.ChannelID = p.ChannelID

		// Gather current channel members
		var members []Member
		h.mu.RLock()
		for _, c := range h.clients {
			if c.ChannelID == p.ChannelID {
				members = append(members, Member{
					UserID:      c.UserID,
					DisplayName: c.DisplayName,
					Speaking:    false,
					Muted:       c.Muted,
					ClientType:  "web",
				})
			}
		}
		h.mu.RUnlock()

		// Include self in the list sent to others
		self := Member{
			UserID:      client.UserID,
			DisplayName: client.DisplayName,
			Speaking:    false,
			Muted:       client.Muted,
			ClientType:  "web",
		}

		client.SendJSON(NewEnvelope(TypeChannelJoined, env.ID, ChannelJoinedPayload{
			ChannelID: p.ChannelID,
			Members:   append(members, self),
		}))

		// Notify other members
		if oldCh != "" && oldCh != p.ChannelID {
			h.BroadcastToChannel(oldCh,
				NewEnvelope(TypePresenceUpdate, "", PresenceUpdatePayload{ChannelID: oldCh}))
		}
		h.BroadcastToChannelExcept(p.ChannelID, client.UserID,
			NewEnvelope(TypePresenceUpdate, "", PresenceUpdatePayload{
				ChannelID: p.ChannelID,
				Members:   append(members, self),
			}))

	case TypeLeaveChannel:
		oldCh := client.ChannelID
		client.ChannelID = ""
		client.SendJSON(NewEnvelope(TypeChannelLeft, env.ID, nil))
		if oldCh != "" {
			h.broadcastPresence(oldCh)
		}

	case TypeStartSpeaking:
		if client.ChannelID != "" {
			h.BroadcastToChannelExcept(client.ChannelID, client.UserID,
				NewEnvelope(TypeUserSpeaking, "", UserSpeakingPayload{UserID: client.UserID, Speaking: true}))
		}

	case TypeStopSpeaking:
		if client.ChannelID != "" {
			h.BroadcastToChannelExcept(client.ChannelID, client.UserID,
				NewEnvelope(TypeUserSpeaking, "", UserSpeakingPayload{UserID: client.UserID, Speaking: false}))
		}

	case TypePing:
		client.SendJSON(NewEnvelope(TypePong, env.ID, nil))

	case TypeSetMute:
		client.Muted = !client.Muted
		if client.ChannelID != "" {
			h.broadcastPresence(client.ChannelID)
		}

	case TypeSetDeafen:
		client.Deafened = !client.Deafened
		if client.ChannelID != "" {
			h.broadcastPresence(client.ChannelID)
		}

	case TypeChatMessage:
		if client.ChannelID != "" {
			var chatPayload ChatMessagePayload
			json.Unmarshal(env.Payload, &chatPayload)
			chatPayload.SenderID = client.UserID
			chatPayload.SenderName = client.DisplayName
			chatPayload.ChannelID = client.ChannelID
			h.BroadcastToChannel(client.ChannelID,
				NewEnvelope(TypeChatMessage, env.ID, chatPayload))
		}

	case TypeSDPOffer:
		if client.ChannelID != "" {
			var p SDPSignalPayload
			json.Unmarshal(env.Payload, &p)
			p.SenderID = client.UserID
			h.BroadcastToChannelExcept(client.ChannelID, client.UserID,
				NewEnvelope(TypeSDPOffer, env.ID, p))
		}

	case TypeSDPAnswer:
		var p SDPSignalPayload
		json.Unmarshal(env.Payload, &p)
		targetUserID := p.SenderID       // save before overwriting
		p.SenderID = client.UserID       // overwrite so receiver knows who answered
		for _, target := range h.GetClientsByUser(targetUserID) {
			if target != nil {
				target.SendJSON(NewEnvelope(TypeSDPAnswer, env.ID, p))
			}
		}

	case TypeICECandidate:
		if client.ChannelID != "" {
			var p ICECandidatePayload
			json.Unmarshal(env.Payload, &p)
			p.SenderID = client.UserID
			h.BroadcastToChannelExcept(client.ChannelID, client.UserID,
				NewEnvelope(TypeICECandidate, env.ID, p))
		}
	}
}

func (h *Hub) handleBinary(client *Client, data []byte) {
	if client.ChannelID == "" {
		return
	}
	// Relay binary audio to all other clients in the same channel
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c.ChannelID == client.ChannelID && c.UserID != client.UserID && !c.Deafened {
			c.SendBinary(data)
		}
	}
}
