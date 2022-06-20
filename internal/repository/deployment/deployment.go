package deployment

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=deployment -destination=mock_deployment.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*appsv1.Deployment, error)
	Create(ctx context.Context, deployment *appsv1.Deployment) error
}

type CRRepository struct {
	client client.Client
}

func NewDeploymentRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*appsv1.Deployment, error) {
	deployment := appsv1.Deployment{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &deployment)
	return &deployment, err
}

func (r *CRRepository) Create(ctx context.Context, deployment *appsv1.Deployment) error {
	return r.client.Create(ctx, deployment)
}
