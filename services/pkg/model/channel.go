package model

import "time"

type Channel struct {
	ID              string     `json:"id"`
	ParentID        *string    `json:"parent_id,omitempty"`
	Name            string     `json:"name"`
	Description     string     `json:"description,omitempty"`
	SortOrder       int        `json:"sort_order"`
	MaxUsers        int        `json:"max_users"` // -1 = unlimited
	CodecQuality    string     `json:"codec_quality"`
	PasswordHash    string     `json:"-"`
	IsTemporary     bool       `json:"is_temporary"`
	TemporaryTTLMin int        `json:"temporary_ttl_min,omitempty"`
	CreatedBy       string     `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	// Computed
	Children    []*Channel `json:"children,omitempty"`
	MemberCount int        `json:"member_count"`
	HasPassword bool       `json:"has_password"`
}

type CreateChannelRequest struct {
	Name            string `json:"name"`
	ParentID        *string `json:"parent_id,omitempty"`
	Description     string `json:"description,omitempty"`
	MaxUsers        int    `json:"max_users,omitempty"`
	CodecQuality    string `json:"codec_quality,omitempty"`
	Password        string `json:"password,omitempty"`
	IsTemporary     bool   `json:"is_temporary,omitempty"`
	TemporaryTTLMin int    `json:"temporary_ttl_min,omitempty"`
}

type UpdateChannelRequest struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	SortOrder       *int    `json:"sort_order,omitempty"`
	MaxUsers        *int    `json:"max_users,omitempty"`
	CodecQuality    *string `json:"codec_quality,omitempty"`
	Password        *string `json:"password,omitempty"`
	IsTemporary     *bool   `json:"is_temporary,omitempty"`
	TemporaryTTLMin *int    `json:"temporary_ttl_min,omitempty"`
}

type JoinChannelRequest struct {
	Password string `json:"password,omitempty"`
}

type ChannelMembership struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	ChannelID   string     `json:"channel_id"`
	DisplayName string     `json:"display_name"`
	ClientType  string     `json:"client_type"` // "web" | "embedded"
	DeviceID    *string    `json:"device_id,omitempty"`
	JoinedAt    time.Time  `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at,omitempty"`
}
