package osbuild

import (
	"context"
	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=osbuild -destination=mock_osbuild.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuild, error)
	Create(ctx context.Context, osBuild *v1alpha1.OSBuild) error
}

type CRRepository struct {
	client client.Client
}

func NewOSBuildRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuild, error) {
	osBuild := v1alpha1.OSBuild{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &osBuild)
	return &osBuild, err
}

func (r *CRRepository) Create(ctx context.Context, osBuild *v1alpha1.OSBuild) error {
	return r.client.Create(ctx, osBuild)
}
