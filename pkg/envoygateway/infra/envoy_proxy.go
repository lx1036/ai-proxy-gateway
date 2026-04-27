package infra

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/lx1036/gateway/pkg/envoygateway/ir"
)

func (infra *Infra) CreateOrUpdateProxyInfra(ctx context.Context, infraIR *ir.Infra)  {

	render, err := NewResourceRender(ctx, infraIR)

	err = infra.createOrUpdate(ctx, render)
}

func (infra *Infra) createOrUpdate(ctx context.Context, render *ResourceRender) error {

	err := infra.createOrUpdateDeployment(ctx, render)
	if err != nil {
		return fmt.Errorf("failed to create or update deployment %s/%s: %w", render.Namespace(), render.Name(), err)
	}

	return nil
}

func (infra *Infra) createOrUpdateDeployment(ctx context.Context, render *ResourceRender) error {


	deployment, err := render.Deployment()




	old := &appsv1.Deployment{}

	err = infra.InfraClient.Get(ctx, types.NamespacedName{
		Namespace: "",
		Name:      "",
	}, old)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// INFO: 1. create deployment
			return infra.InfraClient.ServerSideApply(ctx, deployment)
		}

		return err
	}

	if !equality.Semantic.DeepEqual(old.Spec.Selector, deployment.Spec.Selector) {
	}


	// INFO: 2. update deployment
	return infra.InfraClient.ServerSideApply(ctx, deployment)
}
