package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voxmesh/pkg/model"
)

type ChannelRepo struct {
	pool *pgxpool.Pool
}

func NewChannelRepo(pool *pgxpool.Pool) *ChannelRepo {
	return &ChannelRepo{pool: pool}
}

func (r *ChannelRepo) Create(ctx context.Context, req model.CreateChannelRequest, createdBy string) (*model.Channel, error) {
	var ch model.Channel
	err := r.pool.QueryRow(ctx,
		`INSERT INTO channels (id, parent_id, name, description, max_users, codec_quality, password_hash, is_temporary, temporary_ttl_min, created_by)
		 VALUES (replace(gen_random_uuid()::text, '-', ''), $1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, parent_id, name, description, sort_order, max_users, codec_quality, is_temporary, temporary_ttl_min, created_by, created_at, updated_at`,
		req.ParentID, req.Name, req.Description, req.MaxUsers, req.CodecQuality, req.Password, req.IsTemporary, req.TemporaryTTLMin, createdBy,
	).Scan(&ch.ID, &ch.ParentID, &ch.Name, &ch.Description, &ch.SortOrder, &ch.MaxUsers, &ch.CodecQuality, &ch.IsTemporary, &ch.TemporaryTTLMin, &ch.CreatedBy, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}
	ch.HasPassword = req.Password != ""
	return &ch, nil
}

func (r *ChannelRepo) FindByID(ctx context.Context, id string) (*model.Channel, error) {
	var ch model.Channel
	err := r.pool.QueryRow(ctx,
		`SELECT id, parent_id, name, COALESCE(description, '') as description, sort_order, max_users, codec_quality, COALESCE(password_hash,'') as password_hash, is_temporary, temporary_ttl_min, COALESCE(created_by::text,''), created_at, updated_at
		 FROM channels WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&ch.ID, &ch.ParentID, &ch.Name, &ch.Description, &ch.SortOrder, &ch.MaxUsers, &ch.CodecQuality, &ch.PasswordHash, &ch.IsTemporary, &ch.TemporaryTTLMin, &ch.CreatedBy, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("find channel: %w", err)
	}
	ch.HasPassword = ch.PasswordHash != ""
	return &ch, nil
}

func (r *ChannelRepo) FindByParent(ctx context.Context, parentID *string) ([]*model.Channel, error) {
	var rows pgx.Rows
	var err error
	if parentID == nil {
		rows, err = r.pool.Query(ctx,
			`SELECT id, parent_id, name, COALESCE(description, '') as description, sort_order, max_users, codec_quality, COALESCE(password_hash,''), is_temporary, temporary_ttl_min, COALESCE(created_by::text,''), created_at, updated_at
			 FROM channels WHERE parent_id IS NULL AND deleted_at IS NULL ORDER BY sort_order, name`)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, parent_id, name, COALESCE(description, '') as description, sort_order, max_users, codec_quality, COALESCE(password_hash,''), is_temporary, temporary_ttl_min, COALESCE(created_by::text,''), created_at, updated_at
			 FROM channels WHERE parent_id = $1 AND deleted_at IS NULL ORDER BY sort_order, name`, *parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := make([]*model.Channel, 0)
	for rows.Next() {
		var ch model.Channel
		if err := rows.Scan(&ch.ID, &ch.ParentID, &ch.Name, &ch.Description, &ch.SortOrder, &ch.MaxUsers, &ch.CodecQuality, &ch.PasswordHash, &ch.IsTemporary, &ch.TemporaryTTLMin, &ch.CreatedBy, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		ch.HasPassword = ch.PasswordHash != ""
		channels = append(channels, &ch)
	}
	return channels, nil
}

func (r *ChannelRepo) CountMembers(ctx context.Context, channelID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM channel_memberships WHERE channel_id = $1 AND left_at IS NULL`, channelID).Scan(&count)
	return count, err
}

func (r *ChannelRepo) Join(ctx context.Context, userID, channelID, clientType string, deviceID *string) error {
	// Check for existing active membership
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM channel_memberships WHERE user_id = $1 AND channel_id = $2 AND left_at IS NULL`,
		userID, channelID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("already in channel")
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO channel_memberships (user_id, channel_id, client_type, device_id) VALUES ($1, $2, $3, $4)`,
		userID, channelID, clientType, deviceID)
	return err
}

func (r *ChannelRepo) Leave(ctx context.Context, userID, channelID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE channel_memberships SET left_at = $1 WHERE user_id = $2 AND channel_id = $3 AND left_at IS NULL`, time.Now(), userID, channelID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("not in channel")
	}
	return nil
}

func (r *ChannelRepo) GetMembers(ctx context.Context, channelID string) ([]model.ChannelMembership, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cm.id, cm.user_id, cm.channel_id, COALESCE(u.display_name, u.username) AS display_name,
			cm.client_type, COALESCE(cm.device_id,''), cm.joined_at, cm.left_at
		 FROM channel_memberships cm
		 JOIN users u ON cm.user_id = u.id
		 WHERE cm.channel_id = $1 AND cm.left_at IS NULL`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]model.ChannelMembership, 0)
	for rows.Next() {
		var m model.ChannelMembership
		if err := rows.Scan(&m.ID, &m.UserID, &m.ChannelID, &m.DisplayName, &m.ClientType, &m.DeviceID, &m.JoinedAt, &m.LeftAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *ChannelRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE channels SET deleted_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *ChannelRepo) Update(ctx context.Context, id string, req model.UpdateChannelRequest) (*model.Channel, error) {
	ch, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		ch.Name = *req.Name
	}
	if req.Description != nil {
		ch.Description = *req.Description
	}
	if req.MaxUsers != nil {
		ch.MaxUsers = *req.MaxUsers
	}
	if req.CodecQuality != nil {
		ch.CodecQuality = *req.CodecQuality
	}
	if req.Password != nil {
		ch.PasswordHash = *req.Password
		ch.HasPassword = *req.Password != ""
	}
	if req.SortOrder != nil {
		ch.SortOrder = *req.SortOrder
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE channels SET name=$1, description=$2, max_users=$3, codec_quality=$4, password_hash=$5, sort_order=$6, updated_at=$7 WHERE id=$8`,
		ch.Name, ch.Description, ch.MaxUsers, ch.CodecQuality, ch.PasswordHash, ch.SortOrder, time.Now(), id)
	if err != nil {
		return nil, err
	}
	return ch, nil
}
