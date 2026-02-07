package app

import (
	"context"

	"github.com/mikkel-kaj/iceberg/internal/domain"
	"github.com/mikkel-kaj/iceberg/internal/ports"
)

type ServerUseCase struct {
	repo ports.ServerRepository
}

func NewServerUseCase(repo ports.ServerRepository) *ServerUseCase {
	return &ServerUseCase{repo: repo}
}

func (u *ServerUseCase) Register(ctx context.Context, s domain.Server) error {
	return u.repo.CreateServer(ctx, s)
}

func (u *ServerUseCase) Get(ctx context.Context, name string) (*domain.Server, error) {
	return u.repo.GetServerByName(ctx, name)
}

func (u *ServerUseCase) List(ctx context.Context) ([]domain.Server, error) {
	return u.repo.ListServers(ctx)
}

func (u *ServerUseCase) Delete(ctx context.Context, name string) error {
	return u.repo.DeleteServerByName(ctx, name)
}
