package types

type Pod interface {
	GetPod() *backend.Pod
	GetMetrics() *backendmetrics.MetricsState
	String() string
	//Get(string) (datalayer.Cloneable, bool)
	//Put(string, datalayer.Cloneable)
	Keys() []string
}

type ProfileRunResult struct {
	TargetPods []Pod
}
