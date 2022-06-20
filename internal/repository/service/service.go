package service

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=service -destination=mock_service.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*corev1.Service, error)
	Create(ctx context.Context, service *corev1.Service) error
}

type CRRepository struct {
	client client.Client
}

func NewServiceRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*corev1.Service, error) {
	service := corev1.Service{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &service)
	return &service, err
}

func (r *CRRepository) Create(ctx context.Context, service *corev1.Service) error {
	return r.client.Create(ctx, service)
}
