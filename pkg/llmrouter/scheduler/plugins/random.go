package plugins

import (
	"github.com/lx1036/gateway/pkg/llmrouter/datastore"
	"github.com/lx1036/gateway/pkg/llmrouter/scheduler/framework"
	"k8s.io/apimachinery/pkg/runtime"
	"math/rand"
	"time"
)

const RandomPluginName = "random"

type Random struct {
	name string
	rng  *rand.Rand
}

func NewRandom(pluginArg runtime.RawExtension) *Random {

	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	return &Random{
		name: RandomPluginName,
		rng:  rng,
	}
}

func (r *Random) Name() string {
	return r.name
}

// Score assigns random scores to pods within the range [0, 100]
func (r *Random) Score(ctx *framework.SchedulerContext, pods []*datastore.PodInfo) map[*datastore.PodInfo]int {
	scoreResults := make(map[*datastore.PodInfo]int)
	// Assign random scores between 0 and 100 to each pod
	for _, pod := range pods {
		score := r.rng.Intn(101) // Generate random number between 0-100 (inclusive)
		scoreResults[pod] = score
	}

	return scoreResults
}
