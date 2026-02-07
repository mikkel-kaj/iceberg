package db

import (
	"context"
	"database/sql"
	"errors"

	sqlcdb "github.com/mikkel-kaj/iceberg/internal/adapters/db/sqlc"
	"github.com/mikkel-kaj/iceberg/internal/domain"
)

type Repository struct {
	q *sqlcdb.Queries
}

func NewRepository(q *sqlcdb.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) CreateServer(ctx context.Context, s domain.Server) error {
	return r.q.CreateServer(ctx, sqlcdb.CreateServerParams{
		Name:              s.Name,
		TailscaleHostname: s.TailscaleHostname,
		HetznerID:         s.HetznerID,
		IP:                s.IP,
		AgentToken:        s.AgentToken,
		SSHKeyID:          s.SSHKeyID,
		FirewallID:        s.FirewallID,
	})
}

func (r *Repository) GetServerByName(ctx context.Context, name string) (*domain.Server, error) {
	s, err := r.q.GetServerByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.Server{Name: s.Name, TailscaleHostname: s.TailscaleHostname, HetznerID: s.HetznerID, IP: s.Ip, AgentToken: s.AgentToken, SSHKeyID: s.SshKeyID, FirewallID: s.FirewallID, CreatedAt: s.CreatedAt}, nil
}

func (r *Repository) ListServers(ctx context.Context) ([]domain.Server, error) {
	rows, err := r.q.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Server, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.Server{Name: s.Name, TailscaleHostname: s.TailscaleHostname, HetznerID: s.HetznerID, IP: s.Ip, AgentToken: s.AgentToken, SSHKeyID: s.SshKeyID, FirewallID: s.FirewallID, CreatedAt: s.CreatedAt})
	}
	return out, nil
}

func (r *Repository) DeleteServerByName(ctx context.Context, name string) error {
	return r.q.DeleteServerByName(ctx, name)
}

func (r *Repository) UpsertService(ctx context.Context, s domain.Service) error {
	return r.q.UpsertService(ctx, sqlcdb.UpsertServiceParams{Name: s.Name, ServerName: s.Server, Domain: s.Domain, Status: s.Status, SpecJson: s.SpecJSON})
}

func (r *Repository) GetServiceByName(ctx context.Context, name string) (*domain.Service, error) {
	s, err := r.q.GetServiceByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.Service{Name: s.Name, Server: s.ServerName, Domain: s.Domain, Status: s.Status, SpecJSON: s.SpecJson, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}, nil
}

func (r *Repository) ListServices(ctx context.Context) ([]domain.Service, error) {
	rows, err := r.q.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Service, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.Service{Name: s.Name, Server: s.ServerName, Domain: s.Domain, Status: s.Status, SpecJSON: s.SpecJson, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt})
	}
	return out, nil
}

func (r *Repository) DeleteServiceByName(ctx context.Context, name string) error {
	return r.q.DeleteServiceByName(ctx, name)
}
