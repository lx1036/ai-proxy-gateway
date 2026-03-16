package collector

import (
	"fmt"
	"context"
)

type Collector struct {
}

func (c *Collector) Collect(ctx context.Context, endpoint string) {

	url := fmt.Sprintf("http://%s/metrics", endpoint)

	allMetrics, err = c.fetchAllMetricsFromURL(ctx, url)

}
