package osbuildenvconfig

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
)

//go:generate mockgen -package=osbuildenvconfig -destination=mock_osbuildenvconfig.go . Repository
type Repository interface {
	Read(ctx context.Context, name string) (*v1alpha1.OSBuildEnvConfig, error)
	Patch(ctx context.Context, old, new *v1alpha1.OSBuildEnvConfig) error
}

type CRRepository struct {
	client client.Client
}

func NewOSBuildEnvConfigRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string) (*v1alpha1.OSBuildEnvConfig, error) {
	osBuildEnvConfig := v1alpha1.OSBuildEnvConfig{}
	err := r.client.Get(ctx, client.ObjectKey{Name: name}, &osBuildEnvConfig)
	return &osBuildEnvConfig, err
}

func (r *CRRepository) Patch(ctx context.Context, old, new *v1alpha1.OSBuildEnvConfig) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}
