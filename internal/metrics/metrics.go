package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	ManagedConfigMapCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_managed_configmap_count",
			Help: "Number of managed configmaps by argo rollout config keeper operator",
		})
	ManagedSecretCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_managed_secret_count",
			Help: "Number of managed secrets by argo rollout config keeper operator",
		})
	DiscoveredConfigMapCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_discovered_configmap_count",
			Help: "Number of discovered configmaps by argo rollout config keeper operator",
		})
	DiscoveredSecretCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_discovered_secret_count",
			Help: "Number of discovered secrets by argo rollout config keeper operator",
		})
	FailuresInConfigMapCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_failures_configmap_count",
			Help: "Number of failures in configmap reconciliation",
		})
	FailuresInSecretCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_failures_secret_count",
			Help: "Number of failures in secret reconciliation",
		})
	ConfigMapReconcileDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "argo_rollout_config_keeper_configmap_reconcile_duration_seconds",
			Help: "Time taken to reconcile configmaps",
		})
	SecretReconcileDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "argo_rollout_config_keeper_secret_reconcile_duration_seconds",
			Help: "Time taken to reconcile secrets",
		})
	OverallReconcileDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "argo_rollout_config_keeper_overall_reconcile_duration_seconds",
			Help: "Time taken to reconcile overall process",
		})
	ManagedConfigMapClusterScopeCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_managed_configmap_clusterscope_count",
			Help: "Number of managed configmaps by argo rollout config keeper cluster scope operator",
		})
	ManagedSecretClusterScopeCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_managed_secret_clusterscope_count",
			Help: "Number of managed secrets by argo rollout config keeper cluster scope operator",
		})
	DiscoveredConfigMapClusterScopeCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_discovered_configmap_clusterscope_count",
			Help: "Number of discovered configmaps by argo rollout config keeper cluster scope operator",
		})
	DiscoveredSecretClusterScopeCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_discovered_secret_clusterscope_count",
			Help: "Number of discovered secrets by argo rollout config keeper cluster scope operator",
		})
	FailuresInConfigMapClusterScopeCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_failures_configmap_clusterscope_count",
			Help: "Number of failures in configmap reconciliation by argo rollout config keeper cluster scope operator",
		})
	FailuresInSecretClusterScopeCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "argo_rollout_config_keeper_failures_secret_clusterscope_count",
			Help: "Number of failures in secret reconciliation by argo rollout config keeper cluster scope operator",
		})
	ConfigMapClusterScopeReconcileDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "argo_rollout_config_keeper_configmap_clusterscope_reconcile_duration_seconds",
			Help: "Time taken to reconcile configmaps by argo rollout config keeper cluster scope operator",
		})
	SecretClusterScopeReconcileDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "argo_rollout_config_keeper_secret_clusterscope_reconcile_duration_seconds",
			Help: "Time taken to reconcile secrets by argo rollout config keeper cluster scope operator",
		})
	OverallClusterScopeReconcileDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "argo_rollout_config_keeper_overall_clusterscope_reconcile_duration_seconds",
			Help: "Time taken to reconcile overall process by argo rollout config keeper cluster scope operator",
		})
)

func init() {
	// Register own metrics
	for _, metric := range registerOwnMetrics() {
		switch metric := metric.(type) {
		case prometheus.Histogram:
			metrics.Registry.MustRegister(metric)
		case prometheus.Gauge:
			metrics.Registry.MustRegister(metric)
		case prometheus.Counter:
			metrics.Registry.MustRegister(metric)
		}
	}
}

func registerOwnMetrics() []interface{} {
	return []interface{}{
		ManagedConfigMapCount,
		ManagedSecretCount,
		DiscoveredConfigMapCount,
		DiscoveredSecretCount,
		FailuresInConfigMapCount,
		FailuresInSecretCount,
		ConfigMapReconcileDuration,
		SecretReconcileDuration,
		OverallReconcileDuration,
		ManagedConfigMapClusterScopeCount,
		ManagedSecretClusterScopeCount,
		DiscoveredConfigMapClusterScopeCount,
		DiscoveredSecretClusterScopeCount,
		FailuresInConfigMapClusterScopeCount,
		FailuresInSecretClusterScopeCount,
		ConfigMapClusterScopeReconcileDuration,
		SecretClusterScopeReconcileDuration,
		OverallClusterScopeReconcileDuration,
	}
}
