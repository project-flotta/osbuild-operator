package secret

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=secret -destination=mock_secret.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*corev1.Secret, error)
	Create(ctx context.Context, secret *corev1.Secret) error
	Delete(ctx context.Context, secret *corev1.Secret) error
}

type CRRepository struct {
	client client.Client
}

func NewSecretRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*corev1.Secret, error) {
	secret := corev1.Secret{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &secret)
	return &secret, err
}

func (r *CRRepository) Create(ctx context.Context, secret *corev1.Secret) error {
	return r.client.Create(ctx, secret)
}

func (r *CRRepository) Delete(ctx context.Context, secret *corev1.Secret) error {
	return r.client.Delete(ctx, secret)
}
