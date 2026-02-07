package ports

import (
	"context"

	"github.com/mikkel-kaj/iceberg/internal/domain"
)

type ServerRepository interface {
	CreateServer(ctx context.Context, s domain.Server) error
	GetServerByName(ctx context.Context, name string) (*domain.Server, error)
	ListServers(ctx context.Context) ([]domain.Server, error)
	DeleteServerByName(ctx context.Context, name string) error
}

type ServiceRepository interface {
	UpsertService(ctx context.Context, s domain.Service) error
	GetServiceByName(ctx context.Context, name string) (*domain.Service, error)
	ListServices(ctx context.Context) ([]domain.Service, error)
	DeleteServiceByName(ctx context.Context, name string) error
}
