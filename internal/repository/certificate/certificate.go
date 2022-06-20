package certificate

import (
	"context"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=certificate -destination=mock_certificate.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*certmanagerv1.Certificate, error)
	Create(ctx context.Context, cettificate *certmanagerv1.Certificate) error
}

type CRRepository struct {
	client client.Client
}

func NewCertificateRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*certmanagerv1.Certificate, error) {
	certificate := certmanagerv1.Certificate{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &certificate)
	return &certificate, err
}

func (r *CRRepository) Create(ctx context.Context, certificate *certmanagerv1.Certificate) error {
	return r.client.Create(ctx, certificate)
}
