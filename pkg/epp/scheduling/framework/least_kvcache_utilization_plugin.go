package framework

type KVCacheUtilizationPkugin struct {
}

func (plugin *KVCacheUtilizationPkugin) Score(pods []Pod) map[Pod]float64 {
	scores := make(map[Pod]float64, len(pods))
	for _, pod := range pods {
		scores[pod] = 1 - pod.GetMetrics().KVCacheUsagePercent
	}

	return scores
}


