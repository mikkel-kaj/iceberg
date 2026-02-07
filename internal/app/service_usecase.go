package app

import (
	"context"

	"github.com/mikkel-kaj/iceberg/internal/domain"
	"github.com/mikkel-kaj/iceberg/internal/ports"
)

type ServiceUseCase struct {
	repo ports.ServiceRepository
}

func NewServiceUseCase(repo ports.ServiceRepository) *ServiceUseCase {
	return &ServiceUseCase{repo: repo}
}

func (u *ServiceUseCase) Upsert(ctx context.Context, s domain.Service) error {
	return u.repo.UpsertService(ctx, s)
}

func (u *ServiceUseCase) Get(ctx context.Context, name string) (*domain.Service, error) {
	return u.repo.GetServiceByName(ctx, name)
}

func (u *ServiceUseCase) List(ctx context.Context) ([]domain.Service, error) {
	return u.repo.ListServices(ctx)
}

func (u *ServiceUseCase) Delete(ctx context.Context, name string) error {
	return u.repo.DeleteServiceByName(ctx, name)
}
