package osbuildconfig

import (
	"context"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/indexer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=osbuildconfig -destination=mock_osbuildconfig.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuildConfig, error)
	PatchStatus(ctx context.Context, osbuildConfig *v1alpha1.OSBuildConfig, patch *client.Patch) error
	ListByOSBuildConfigTemplate(ctx context.Context, templateName string, namespace string) ([]v1alpha1.OSBuildConfig, error)
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

func (r *CRRepository) ListByOSBuildConfigTemplate(ctx context.Context, templateName string, namespace string) ([]v1alpha1.OSBuildConfig, error) {
	configs := v1alpha1.OSBuildConfigList{}
	err := r.client.List(ctx, &configs,
		client.MatchingFields{indexer.ConfigByConfigTemplate: templateName},
		client.InNamespace(namespace),
	)
	if err != nil {
		return nil, err
	}
	return configs.Items, nil
}
