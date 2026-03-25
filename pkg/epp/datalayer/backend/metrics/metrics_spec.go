package metrics

import (
	"fmt"
	"strings"
)

// MetricSpec represents a single metric's specification.
type MetricSpec struct {
	MetricName string
	Labels     map[string]string // Label name -> Label value
}

type MetricMapping struct {
	TotalQueuedRequests  *MetricSpec
	TotalRunningRequests *MetricSpec
	KVCacheUtilization   *MetricSpec
	LoraRequestInfo      *MetricSpec
	CacheConfigInfo      *MetricSpec
}

func NewMetricMapping(queuedStr, runningStr, kvUsageStr, loraReqInfoStr, cacheInfoMetric string) (*MetricMapping, error) {
	queuedSpec, err := stringToMetricSpec(queuedStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing WaitingRequests: %w", err)
	}
	runningSpec, err := stringToMetricSpec(runningStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing RunningRequests: %w", err)
	}
	kvUsageSpec, err := stringToMetricSpec(kvUsageStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing KVCacheUsage: %w", err)
	}
	loraReqInfoSpec, err := stringToMetricSpec(loraReqInfoStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing loraReqInfoStr: %w", err)
	}

	cacheInfoSpec, err := stringToMetricSpec(cacheInfoMetric)
	if err != nil {
		return nil, fmt.Errorf("error parsing cacheInfoMetric: %w", err)
	}

	mapping := &MetricMapping{
		TotalQueuedRequests:  queuedSpec,
		TotalRunningRequests: runningSpec,
		KVCacheUtilization:   kvUsageSpec,
		LoraRequestInfo:      loraReqInfoSpec,
		CacheConfigInfo:      cacheInfoSpec,
	}

	return mapping, nil
}

// stringToMetricSpec converts a string to a MetricSpec.
// Example inputs:
//
//	"metric_name"
//	"metric_name{label1=value1}"
//	"metric_name{label1=value1,label2=value2}"
func stringToMetricSpec(specStr string) (*MetricSpec, error) {
	if specStr == "" {
		return nil, nil // Allow empty strings to represent nil MetricSpecs
	}
	specStr = strings.TrimSpace(specStr)
	metricName := specStr
	labels := make(map[string]string)

	// Check for labels enclosed in curly braces
	start := strings.Index(specStr, "{")
	end := strings.Index(specStr, "}")

	if start != -1 || end != -1 { // If *either* brace is present...
		if start == -1 || end == -1 || end <= start+1 { // ...check that *both* are present and correctly placed.
			return nil, fmt.Errorf("invalid metric spec string: %q, missing or malformed label block", specStr)
		}

		metricName = strings.TrimSpace(specStr[:start])
		labelStr := specStr[start+1 : end]

		// Split into individual label pairs
		labelPairs := strings.Split(labelStr, ",")
		for _, pair := range labelPairs {
			pair = strings.TrimSpace(pair)
			parts := strings.Split(pair, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid label pair: %q in metric spec: %q", pair, specStr)
			}
			labelName := strings.TrimSpace(parts[0])
			labelValue := strings.TrimSpace(parts[1])
			if labelName == "" || labelValue == "" {
				return nil, fmt.Errorf("empty label name or value in pair: %q in metric spec: %q", pair, specStr)
			}
			labels[labelName] = labelValue
		}
		// Check for extra characters after labels
		if end != len(specStr)-1 {
			return nil, fmt.Errorf("invalid characters after label section in: %q", specStr)
		}

	}

	if metricName == "" { // Metric name cannot be empty
		return nil, fmt.Errorf("empty metric name in spec: %q", specStr)
	}

	return &MetricSpec{
		MetricName: metricName,
		Labels:     labels,
	}, nil
}
