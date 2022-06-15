package configmap

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=configmap -destination=mock_configmap.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*corev1.ConfigMap, error)
	Create(ctx context.Context, configMap *corev1.ConfigMap) error
	Patch(ctx context.Context, old, new *corev1.ConfigMap) error
}

type CRRepository struct {
	client client.Client
}

func NewConfigMapRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*corev1.ConfigMap, error) {
	configMap := corev1.ConfigMap{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &configMap)
	return &configMap, err
}

func (r *CRRepository) Create(ctx context.Context, configMap *corev1.ConfigMap) error {
	return r.client.Create(ctx, configMap)
}

func (r *CRRepository) Patch(ctx context.Context, old, new *corev1.ConfigMap) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}
