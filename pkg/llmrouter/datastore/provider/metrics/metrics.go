package metrics

import (
	"fmt"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

// ParseMetricsURL This function refer to aibrix(https://github.com/vllm-project/aibrix/blob/main/pkg/metrics/utils.go)
func ParseMetricsURL(url string) (map[string]*dto.MetricFamily, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch metrics from %s: %v", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}()

	parser := expfmt.NewTextParser(model.UTF8Validation)
	allMetrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing metric families: %v\n", err)
	}
	return allMetrics, nil
}
