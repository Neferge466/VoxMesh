package service

import (
	"context"
	"fmt"
	slogx "github.com/voxmesh/pkg/log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voxmesh/pkg/errors"
	"github.com/voxmesh/pkg/model"
	"golang.org/x/crypto/bcrypt"

	"github.com/voxmesh/channel/internal/repository"
)

type ChannelService struct {
	repo *repository.ChannelRepo
}

func NewChannelService(pool *pgxpool.Pool) *ChannelService {
	return &ChannelService{repo: repository.NewChannelRepo(pool)}
}

func (s *ChannelService) GetChannels(ctx context.Context, parentID *string) ([]*model.Channel, error) {
	channels, err := s.repo.FindByParent(ctx, parentID)
	if err != nil {
		return nil, errors.ErrInternal
	}
	for _, ch := range channels {
		count, _ := s.repo.CountMembers(ctx, ch.ID)
		ch.MemberCount = count
	}
	return channels, nil
}

func (s *ChannelService) GetChannel(ctx context.Context, id string) (*model.Channel, error) {
	ch, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.ErrChannelNotFound
	}
	count, _ := s.repo.CountMembers(ctx, id)
	ch.MemberCount = count
	return ch, nil
}

func (s *ChannelService) CreateChannel(ctx context.Context, req model.CreateChannelRequest, createdBy string) (*model.Channel, error) {
	if req.CodecQuality == "" {
		req.CodecQuality = "high"
	}
	if req.MaxUsers == 0 {
		req.MaxUsers = -1
	}
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.ErrInternal
		}
		req.Password = string(hashed)
	}
	return s.repo.Create(ctx, req, createdBy)
}

func (s *ChannelService) UpdateChannel(ctx context.Context, id string, req model.UpdateChannelRequest) (*model.Channel, error) {
	if req.Password != nil && *req.Password != "" {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		hashStr := string(hashed)
		req.Password = &hashStr
	}
	return s.repo.Update(ctx, id, req)
}

func (s *ChannelService) DeleteChannel(ctx context.Context, id string) error {
	// Prevent deleting channels with children
	children, _ := s.repo.FindByParent(ctx, &id)
	if len(children) > 0 {
		return errors.ErrCannotCreateChild
	}
	return s.repo.Delete(ctx, id)
}

func (s *ChannelService) JoinChannel(ctx context.Context, channelID, userID, clientType string, deviceID *string, password string) error {
	ch, err := s.repo.FindByID(ctx, channelID)
	if err != nil {
		slogx.Info("[channel] JoinChannel: FindByID failed channel=%s err=%v", channelID, err)
		return errors.ErrChannelNotFound
	}
	if ch.HasPassword && ch.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(ch.PasswordHash), []byte(password)); err != nil {
			return errors.ErrChannelLocked
		}
	}
	if ch.MaxUsers > 0 {
		count, _ := s.repo.CountMembers(ctx, channelID)
		if count >= ch.MaxUsers {
			return errors.ErrChannelFull
		}
	}
	if err := s.repo.Join(ctx, userID, channelID, clientType, deviceID); err != nil {
		if err.Error() == "already in channel" {
			return errors.ErrAlreadyInChannel
		}
		slogx.Info("[channel] JoinChannel: repo.Join failed channel=%s user=%s err=%v", channelID, userID, err)
		return errors.ErrInternal
	}
	return nil
}

func (s *ChannelService) LeaveChannel(ctx context.Context, channelID, userID string) error {
	if err := s.repo.Leave(ctx, userID, channelID); err != nil {
		if err.Error() == "not in channel" {
			return errors.ErrNotInChannel
		}
		slogx.Info("[channel] LeaveChannel: repo.Leave failed channel=%s user=%s err=%v", channelID, userID, err)
		return errors.ErrInternal
	}
	return nil
}

func (s *ChannelService) GetMembers(ctx context.Context, channelID string) ([]model.ChannelMembership, error) {
	return s.repo.GetMembers(ctx, channelID)
}

func (s *ChannelService) KickUser(ctx context.Context, channelID, targetUserID, requesterID string) error {
	// Only channel creator or admin can kick
	ch, err := s.repo.FindByID(ctx, channelID)
	if err != nil {
		return errors.ErrChannelNotFound
	}
	if ch.CreatedBy != requesterID {
		return errors.New(41007, "only the channel creator can kick users")
	}
	return s.repo.Leave(ctx, targetUserID, channelID)
}

func (s *ChannelService) MoveUser(ctx context.Context, fromChannelID, toChannelID, userID string) error {
	if err := s.repo.Leave(ctx, userID, fromChannelID); err != nil {
		return fmt.Errorf("leave source channel: %w", err)
	}
	return s.repo.Join(ctx, userID, toChannelID, "web", nil)
}
