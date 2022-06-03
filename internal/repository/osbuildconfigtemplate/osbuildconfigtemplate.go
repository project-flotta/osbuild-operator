package osbuildconfigtemplate

import (
	"context"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=osbuildconfigtemplate -destination=mock_osbuildconfigtemplate.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuildConfigTemplate, error)
}

type CRRepository struct {
	client client.Client
}

func NewOSBuildConfigTemplateRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuildConfigTemplate, error) {
	template := v1alpha1.OSBuildConfigTemplate{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &template)
	return &template, err
}
