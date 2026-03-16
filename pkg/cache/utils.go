package cache

import (
	"k8s.io/klog/v2"
	"strconv"
)

func getPodMetricPort(pod *Pod) int {
	if pod == nil || pod.Labels == nil {
		return defaultMetricPort
	}
	if v, ok := pod.Labels[MetricPortLabel]; ok && v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		} else {
			klog.Warningf("Invalid value for label %s on pod %s/%s: %q. Using default port %d.", MetricPortLabel, pod.Namespace, pod.Name, v, defaultMetricPort)
		}
	}
	return defaultMetricPort
}
