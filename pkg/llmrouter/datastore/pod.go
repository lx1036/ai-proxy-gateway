package datastore

import (
	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
)

type PodInfo struct {
	Pod *corev1.Pod

	// Name of AI inference engine
	engine string


	// TODO: add metrics here
	GPUCacheUsage     float64 // GPU KV-cache usage.
	RequestWaitingNum float64 // Number of requests waiting to be processed.
	RequestRunningNum float64 // Number of requests running.
	// for calculating the average value over the time interval, need to store the results of the last query
	TimeToFirstToken   *dto.Histogram
	TimePerOutputToken *dto.Histogram
	TPOT               float64
	TTFT               float64


}
