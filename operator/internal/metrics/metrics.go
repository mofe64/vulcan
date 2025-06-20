package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Singleton instance
var (
	instance *Metrics
	once     sync.Once
)

type Metrics struct {
	OrgclusterCount     *prometheus.GaugeVec
	OrgprojectCount     *prometheus.GaugeVec
	OrgapplicationCount *prometheus.GaugeVec
	OrgQuotaUsage       *prometheus.GaugeVec
}

var (
	ClustersPerOrg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "vulkan",
			Subsystem: "cluster",
			Name:      "current_total",
			Help:      "Current number of Cluster CRs counted against each Org quota",
		},
		[]string{"org"},
	)
	ProjectsPerOrg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "vulkan",
			Subsystem: "project",
			Name:      "current_total",
			Help:      "Current number of Project CRs counted against each Org quota",
		},
		[]string{"org"},
	)
	ApplicationsPerOrg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "vulkan",
			Subsystem: "application",
			Name:      "current_total",
			Help:      "Current number of Application CRs counted against each Org quota",
		},
		[]string{"org"},
	)
	OrgQuotaUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "vulkan",
			Subsystem: "org",
			Name:      "quota_usage",
			Help:      "Organization quota usage percentage",
		}, []string{"org", "resource_type"},
	)
)

func init() {
	metrics.Registry.MustRegister(ClustersPerOrg, ProjectsPerOrg, ApplicationsPerOrg, OrgQuotaUsage)
}

func IncClusters(org string)     { ClustersPerOrg.WithLabelValues(org).Inc() }
func DecClusters(org string)     { ClustersPerOrg.WithLabelValues(org).Dec() }
func IncProjects(org string)     { ProjectsPerOrg.WithLabelValues(org).Inc() }
func DecProjects(org string)     { ProjectsPerOrg.WithLabelValues(org).Dec() }
func IncApplications(org string) { ApplicationsPerOrg.WithLabelValues(org).Inc() }
func DecApplications(org string) { ApplicationsPerOrg.WithLabelValues(org).Dec() }
func UpdateQuotaUsage(org string, resourceType string, usage float64) {
	OrgQuotaUsage.WithLabelValues(org, resourceType).Set(usage)
}
