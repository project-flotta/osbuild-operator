package osbuild

import (
	"context"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=osbuild -destination=mock_osbuild.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.OSBuild, error)
	Create(ctx context.Context, osBuild *v1alpha1.OSBuild) error
	PatchStatus(ctx context.Context, osbuild *v1alpha1.OSBuild, patch *client.Patch) error
	Patch(ctx context.Context, old, new *v1alpha1.OSBuild) error
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

func (r *CRRepository) PatchStatus(ctx context.Context, osbuild *v1alpha1.OSBuild, patch *client.Patch) error {
	return r.client.Status().Patch(ctx, osbuild, *patch)
}

func (r *CRRepository) Patch(ctx context.Context, old, new *v1alpha1.OSBuild) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}
