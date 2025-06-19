package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Singleton instance
var (
	instance *Metrics
	once     sync.Once
)

type Metrics struct {
	OrgclusterCount     *prometheus.GaugeVec
	OrgapplicationCount *prometheus.GaugeVec
	OrgQuotaUsage       *prometheus.GaugeVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		OrgclusterCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vulkan_org_cluster_count",
			Help: "Number of clusters per organization",
		}, []string{"org"},
		),
		OrgapplicationCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vulkan_org_application_count",
			Help: "Number of applications per organization",
		}, []string{"org"},
		),
		OrgQuotaUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vulcan_org_quota_usage",
				Help: "Organization quota usage percentage",
			},
			[]string{"org", "resource_type"},
		),
	}
}

// UpdateClusterCount updates the metrics for cluster count
func (m *Metrics) UpdateClusterCount(orgName string, count int32) {
	m.OrgclusterCount.WithLabelValues(orgName).Set(float64(count))
}

// UpdateApplicationCount updates the metrics for application count
func (m *Metrics) UpdateApplicationCount(orgName string, count int32) {
	m.OrgapplicationCount.WithLabelValues(orgName).Set(float64(count))
}

func (m *Metrics) UpdateQuotaUsage(orgName string, resourceType string, usage float64) {
	m.OrgQuotaUsage.WithLabelValues(orgName, resourceType).Set(usage)
	switch resourceType {
	case "clusters":
		m.OrgQuotaUsage.WithLabelValues(orgName, "clusters").Set(usage)
	case "applications":
		m.OrgQuotaUsage.WithLabelValues(orgName, "applications").Set(usage)
	}
}

// GetMetrics returns the singleton metrics instance
func GetMetrics() *Metrics {
	once.Do(func() {
		instance = NewMetrics()
	})
	return instance
}
