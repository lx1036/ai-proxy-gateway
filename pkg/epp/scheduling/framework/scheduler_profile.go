package framework

// SchedulerProfile provides a profile configuration for the scheduler which influence routing decisions.
type SchedulerProfile struct {
	filters []Filter
	scorers []*WeightedScorer
	picker  Picker
}
