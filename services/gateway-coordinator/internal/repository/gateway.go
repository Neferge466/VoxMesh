package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voxmesh/pkg/model"
)

type GatewayRepo struct {
	pool *pgxpool.Pool
}

func NewGatewayRepo(pool *pgxpool.Pool) *GatewayRepo {
	return &GatewayRepo{pool: pool}
}

func (r *GatewayRepo) Create(ctx context.Context, gw *model.Gateway) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateways (id, name, status, ip_address, version, capabilities, last_heartbeat_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		gw.ID, gw.Name, gw.Status, gw.IPAddress, gw.Version, gw.Capabilities, time.Now())
	return err
}

func (r *GatewayRepo) FindByID(ctx context.Context, id string) (*model.Gateway, error) {
	var gw model.Gateway
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, status, COALESCE(ip_address,''), COALESCE(version,''), capabilities, last_heartbeat_at, registered_at, updated_at
		 FROM gateways WHERE id = $1`, id,
	).Scan(&gw.ID, &gw.Name, &gw.Status, &gw.IPAddress, &gw.Version, &gw.Capabilities, &gw.LastHeartbeatAt, &gw.RegisteredAt, &gw.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &gw, nil
}

func (r *GatewayRepo) List(ctx context.Context) ([]*model.Gateway, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, status, COALESCE(ip_address,''), COALESCE(version,''), capabilities, last_heartbeat_at, registered_at, updated_at
		 FROM gateways ORDER BY status, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.Gateway
	for rows.Next() {
		var gw model.Gateway
		if err := rows.Scan(&gw.ID, &gw.Name, &gw.Status, &gw.IPAddress, &gw.Version, &gw.Capabilities, &gw.LastHeartbeatAt, &gw.RegisteredAt, &gw.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &gw)
	}
	return list, nil
}

func (r *GatewayRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateways SET status = $1, updated_at = $2 WHERE id = $3`, status, time.Now(), id)
	return err
}

func (r *GatewayRepo) UpdateHeartbeat(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateways SET last_heartbeat_at = $1, updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *GatewayRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM gateways WHERE id = $1`, id)
	return err
}
