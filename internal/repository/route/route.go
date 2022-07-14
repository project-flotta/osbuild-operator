package route

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"

	routev1 "github.com/openshift/api/route/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=route -destination=mock_route.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*routev1.Route, error)
	Create(ctx context.Context, route *routev1.Route) error
}

type CRRepository struct {
	client client.Client
}

func NewRouteRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*routev1.Route, error) {
	route := routev1.Route{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &route)
	return &route, err
}

func (r *CRRepository) Create(ctx context.Context, route *routev1.Route) error {
	return r.client.Create(ctx, route)
}
