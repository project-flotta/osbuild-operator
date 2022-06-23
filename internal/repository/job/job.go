package job

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"

	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=job -destination=mock_job.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*batchv1.Job, error)
	Create(ctx context.Context, job *batchv1.Job) error
}

type CRRepository struct {
	client client.Client
}

func NewJobRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*batchv1.Job, error) {
	job := batchv1.Job{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &job)
	return &job, err
}

func (r *CRRepository) Create(ctx context.Context, job *batchv1.Job) error {
	return r.client.Create(ctx, job)
}
