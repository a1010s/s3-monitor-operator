package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	BucketSizeBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "s3monitor_bucket_size_bytes",
		Help: "Total size of the bucket in bytes.",
	}, []string{"bucket", "namespace"})

	BucketObjectCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "s3monitor_bucket_object_count",
		Help: "Total number of objects in the bucket.",
	}, []string{"bucket", "namespace"})

	LastScrapeSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "s3monitor_last_scrape_success",
		Help: "1 if the last scrape succeeded, 0 otherwise.",
	}, []string{"bucket", "namespace"})
)

func init() {
	ctrlmetrics.Registry.MustRegister(BucketSizeBytes, BucketObjectCount, LastScrapeSuccess)
}
