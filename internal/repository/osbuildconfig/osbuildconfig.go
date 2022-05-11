package osbuildconfig

import (
	"context"
	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=osbuildconfig -destination=mock_osbuildconfig.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuildConfig, error)
	PatchStatus(ctx context.Context, osbuildConfig *v1alpha1.OSBuildConfig, patch *client.Patch) error
}

type CRRepository struct {
	client client.Client
}

func NewOSBuildConfigRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuildConfig, error) {
	osBuildConfig := v1alpha1.OSBuildConfig{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &osBuildConfig)
	return &osBuildConfig, err
}

func (r *CRRepository) PatchStatus(ctx context.Context, osbuildConfig *v1alpha1.OSBuildConfig, patch *client.Patch) error {
	return r.client.Status().Patch(ctx, osbuildConfig, *patch)
}
