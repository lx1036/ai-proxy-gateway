package framework

import "math"

// RequestWaitingQueuePlugin
type RequestWaitingQueuePlugin struct {
}

func (plugin *RequestWaitingQueuePlugin) Score(pods []Pod) map[Pod]float64 {
	minQueueSize := math.MaxInt
	maxQueueSize := math.MinInt
	// Iterate through the remaining pods to find min and max
	for _, pod := range pods {
		queueSize := pod.GetMetrics().WaitingQueueSize
		if queueSize < minQueueSize {
			minQueueSize = queueSize
		}
		if queueSize > maxQueueSize {
			maxQueueSize = queueSize
		}
	}

	scores := make(map[Pod]float64, len(pods))
	for _, pod := range pods {
		if maxQueueSize-minQueueSize == 0 {
			scores[pod] = 1
			continue
		}

		// INFO: WaitingQueueSize 最小，分数最高
		scores[pod] = float64(maxQueueSize-pod.GetMetrics().WaitingQueueSize) / float64(maxQueueSize-minQueueSize)
	}

	return scores
}
