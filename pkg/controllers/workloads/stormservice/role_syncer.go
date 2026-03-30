package stormservice

import (
	"context"

	orchestrationv1alpha1 "github.com/vllm-project/aibrix/api/orchestration/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatefulRoleSyncer struct {
	cli client.Client
}

func (s *StatefulRoleSyncer) Scale(ctx context.Context, roleSet *orchestrationv1alpha1.RoleSet) {

}
