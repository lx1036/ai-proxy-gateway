package least_request

type leastRequestRouter struct {
}

func (r *leastRequestRouter) Route() {

	targetPod := r.selectTargetPodWithLeastRequestCount(readyPods)

}
