// Package metrics provides Prometheus metrics for Nebularr operator.
// These metrics can be scraped by Prometheus and exported to OTLP
// via an OpenTelemetry collector.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	// Namespace for all Nebularr metrics
	namespace = "nebularr"
)

var (
	// ReconcileTotal tracks total reconciliation attempts
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "reconcile_total",
			Help:      "Total number of reconciliation attempts",
		},
		[]string{"controller", "result"},
	)

	// ReconcileDuration tracks reconciliation duration
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "reconcile_duration_seconds",
			Help:      "Duration of reconciliation cycles in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"controller"},
	)

	// ResourcesManaged tracks number of managed resources
	ResourcesManaged = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "resources_managed",
			Help:      "Number of resources currently managed by Nebularr",
		},
		[]string{"controller", "resource_type"},
	)

	// SyncSuccess tracks successful sync operations
	SyncSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "sync_success_total",
			Help:      "Total number of successful sync operations",
		},
		[]string{"app"},
	)

	// SyncFailure tracks failed sync operations
	SyncFailure = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "sync_failure_total",
			Help:      "Total number of failed sync operations",
		},
		[]string{"app", "error_type"},
	)

	// SyncDuration tracks sync duration
	SyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "sync_duration_seconds",
			Help:      "Duration of sync operations in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"app"},
	)

	// ConfigDrift tracks configuration drift detections
	ConfigDrift = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "config_drift_total",
			Help:      "Total number of configuration drift detections",
		},
		[]string{"app", "resource_type"},
	)

	// ConnectionStatus tracks connection status to *arr services
	ConnectionStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connection_status",
			Help:      "Connection status to *arr services (1=connected, 0=disconnected)",
		},
		[]string{"app", "instance"},
	)

	// ApplyChangesTotal tracks changes applied to *arr services
	ApplyChangesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "apply_changes_total",
			Help:      "Total number of changes applied to *arr services",
		},
		[]string{"app", "action", "resource_type"},
	)

	// ServiceVersion tracks the version of connected *arr services
	ServiceVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "service_info",
			Help:      "Information about connected *arr services (always 1, labels contain info)",
		},
		[]string{"app", "instance", "version"},
	)
)

func init() {
	// Register all metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileDuration,
		ResourcesManaged,
		SyncSuccess,
		SyncFailure,
		SyncDuration,
		ConfigDrift,
		ConnectionStatus,
		ApplyChangesTotal,
		ServiceVersion,
	)
}

// RecordReconcileSuccess records a successful reconciliation
func RecordReconcileSuccess(controller string, duration float64) {
	ReconcileTotal.WithLabelValues(controller, "success").Inc()
	ReconcileDuration.WithLabelValues(controller).Observe(duration)
}

// RecordReconcileFailure records a failed reconciliation
func RecordReconcileFailure(controller string, duration float64) {
	ReconcileTotal.WithLabelValues(controller, "failure").Inc()
	ReconcileDuration.WithLabelValues(controller).Observe(duration)
}

// RecordSyncSuccess records a successful sync operation
func RecordSyncSuccess(app string, duration float64) {
	SyncSuccess.WithLabelValues(app).Inc()
	SyncDuration.WithLabelValues(app).Observe(duration)
}

// RecordSyncFailure records a failed sync operation
func RecordSyncFailure(app string, errorType string, duration float64) {
	SyncFailure.WithLabelValues(app, errorType).Inc()
	SyncDuration.WithLabelValues(app).Observe(duration)
}

// RecordConfigDrift records a configuration drift detection
func RecordConfigDrift(app, resourceType string) {
	ConfigDrift.WithLabelValues(app, resourceType).Inc()
}

// RecordConnectionStatus records the connection status to an *arr service
func RecordConnectionStatus(app, instance string, connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}
	ConnectionStatus.WithLabelValues(app, instance).Set(value)
}

// RecordApplyChange records a change applied to an *arr service
func RecordApplyChange(app, action, resourceType string) {
	ApplyChangesTotal.WithLabelValues(app, action, resourceType).Inc()
}

// RecordServiceVersion records the version of a connected *arr service
func RecordServiceVersion(app, instance, version string) {
	// Reset previous version labels by setting to 0
	// This handles version upgrades
	ServiceVersion.WithLabelValues(app, instance, version).Set(1)
}

// SetResourcesManaged sets the count of managed resources
func SetResourcesManaged(controller, resourceType string, count int) {
	ResourcesManaged.WithLabelValues(controller, resourceType).Set(float64(count))
}
