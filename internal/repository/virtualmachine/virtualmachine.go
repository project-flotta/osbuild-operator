package virtualmachine

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"

	kubevirtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=virtualmachine -destination=mock_virtualmachine.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*kubevirtv1.VirtualMachine, error)
	Create(ctx context.Context, virtualmachine *kubevirtv1.VirtualMachine) error
}

type CRRepository struct {
	client client.Client
}

func NewVirtualMachineRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*kubevirtv1.VirtualMachine, error) {
	virtualmachine := kubevirtv1.VirtualMachine{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &virtualmachine)
	return &virtualmachine, err
}

func (r *CRRepository) Create(ctx context.Context, virtualmachine *kubevirtv1.VirtualMachine) error {
	return r.client.Create(ctx, virtualmachine)
}
