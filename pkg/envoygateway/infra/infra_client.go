package infra

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InfraClient struct {
	client.Client
}

func NewInfraClient(k8sClient client.Client) *InfraClient {
	return &InfraClient{
		Client: k8sClient,
	}
}

func (infraClient *InfraClient) ServerSideApply(ctx context.Context, obj client.Object) error {
	err := infraClient.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("envoy-gateway"))
	if err != nil {
		return fmt.Errorf("failed to create/update resource with server-side apply for obj: %v: %w", obj, err)
	}

	return nil
}
