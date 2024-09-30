package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/run-ai/argo-rollout-config-keeper/internal/tools"

	"github.com/go-logr/logr"
	goversion "github.com/hashicorp/go-version"
	keeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
	"github.com/run-ai/argo-rollout-config-keeper/internal/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ArgoRolloutConfigKeeperLabels struct {
	AppLabel        string
	AppVersionLabel string
}

type ArgoRolloutConfigKeeperCommon struct {
	client.Client
	Scheme        *runtime.Scheme
	Logger        logr.Logger
	Labels        *ArgoRolloutConfigKeeperLabels
	FinalizerName string
}

func (r *ArgoRolloutConfigKeeperCommon) ReconcileConfigMaps(ctx context.Context, namespace string, labelSelector map[string]string, ignoredNamespaces map[string]bool) error {
	defer func() {
		metrics.ConfigMapReconcileDuration.Observe(time.Since(time.Now()).Seconds())
	}()
	metrics.ManagedConfigMapCount.Set(0)

	if configMaps, err := r.listConfigMaps(ctx, namespace, labelSelector); err != nil {
		return err
	} else {
		metrics.DiscoveredConfigMapCount.Set(float64(len(configMaps.Items)))
		if configMaps.Items != nil {
			for _, c := range configMaps.Items {
				r.Logger.Info(fmt.Sprintf("configmap, name: %s", c.Name))
				if _, ok := ignoredNamespaces[c.Namespace]; ok {
					r.Logger.Info(fmt.Sprintf("skipping %s configmap, reason: namespace is ignored", c.Name))
					continue
				}
				if c.Finalizers != nil {
					// check if the finalizer is in the finalizers list
					finalizerFullName, havingManagedFinalizer := tools.ContainsString(c.GetFinalizers(), r.FinalizerName)

					if havingManagedFinalizer {
						metrics.ManagedConfigMapCount.Inc()
						if err := r.finalizerOperation(ctx, &c, finalizerFullName); err != nil {
							return err
						}

						if latestVersion, err := r.getLatestVersionOfConfig(ctx, strings.Split(finalizerFullName, "/")[1], &c); err != nil {
							r.Logger.Error(err, "unable to get latest version of configmap")
							return err
						} else {
							if err := r.ignoreExtraneousOperation(ctx, &c, latestVersion); err != nil {
								return err
							}
						}
					} else {
						r.Logger.Info(fmt.Sprintf("skipping %s configmap, reason: no manageable finalizer", c.Name))
					}
					continue
				}
				r.Logger.Info(fmt.Sprintf("skipping %s configmap, reason: no finalizers", c.Name))
			}
		} else {
			if namespace != "" {
				r.Logger.Info(fmt.Sprintf("no configmaps found in %s namespace", namespace))
			} else {
				r.Logger.Info("no configmaps found")
			}
		}
	}

	return nil
}

func (r *ArgoRolloutConfigKeeperCommon) ReconcileSecrets(ctx context.Context, namespace string, labelSelector map[string]string, ignoredNamespaces map[string]bool) error {
	defer func() {
		metrics.SecretReconcileDuration.Observe(time.Since(time.Now()).Seconds())
	}()
	metrics.ManagedSecretCount.Set(0)

	if secrets, err := r.listSecrets(ctx, namespace, labelSelector); err != nil {
		return err
	} else {
		metrics.DiscoveredSecretCount.Set(float64(len(secrets.Items)))
		if secrets.Items != nil {
			for _, s := range secrets.Items {
				r.Logger.Info(fmt.Sprintf("secret, name: %s", s.Name))
				if _, ok := ignoredNamespaces[s.Namespace]; ok {
					r.Logger.Info(fmt.Sprintf("skipping %s secret, reason: namespace is ignored", s.Name))
					continue
				}

				if s.Finalizers != nil {
					finalizerFullName, havingManagedFinalizer := tools.ContainsString(s.GetFinalizers(), r.FinalizerName)

					if havingManagedFinalizer {
						metrics.ManagedSecretCount.Inc()
						if err := r.finalizerOperation(ctx, &s, finalizerFullName); err != nil {
							return err
						}

						if latestVersion, err := r.getLatestVersionOfConfig(ctx, strings.Split(finalizerFullName, "/")[1], &s); err != nil {
							r.Logger.Error(err, "unable to get latest version of secret")
							return err
						} else {
							if err := r.ignoreExtraneousOperation(ctx, &s, latestVersion); err != nil {
								return err
							}
						}
					} else {
						r.Logger.Info(fmt.Sprintf("skipping %s secret, reason: no manageable finalizer", s.Name))
					}
					continue
				}
				r.Logger.Info(fmt.Sprintf("skipping %s secret, reason: no finalizers", s.Name))
			}
		} else {
			if namespace != "" {
				r.Logger.Info(fmt.Sprintf("no secrets found in %s namespace", namespace))
			} else {
				r.Logger.Info("no secrets found")
			}
		}
	}

	return nil
}

func (r *ArgoRolloutConfigKeeperCommon) finalizerOperation(ctx context.Context, T interface{}, finalizer string) error {
	// Check the type of the object
	switch t := T.(type) {
	case *corev1.ConfigMap:
		r.Logger.Info(fmt.Sprintf("finalizer operation on configmap object, name: %s", t.Name))
		inUse, err := r.checkIfFinalizerInUse(ctx, t.Namespace, strings.Split(finalizer, "/")[1], t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			return err
		}
		if !inUse {
			r.Logger.Info(fmt.Sprintf("removing finalizer from configmap, name: %s, reason: finalizer not in use", t.Name))
			t.ObjectMeta.Finalizers = tools.RemoveString(t.ObjectMeta.Finalizers, finalizer)
			err = r.Update(ctx, t)
			if err != nil {
				r.Logger.Error(err, "unable to remove finalizer from configmap", "name", t.Name)
			}
			metrics.ManagedConfigMapCount.Dec()
		}
		return err
	case *corev1.Secret:
		r.Logger.Info(fmt.Sprintf("finalizer operation on secret object, name: %s", t.Name))
		inUse, err := r.checkIfFinalizerInUse(ctx, t.Namespace, strings.Split(finalizer, "/")[1], t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			return err
		}
		if !inUse {
			r.Logger.Info(fmt.Sprintf("removing finalizer from secret, name: %s, reason: finalizer not in use", t.Name))
			t.ObjectMeta.Finalizers = tools.RemoveString(t.ObjectMeta.Finalizers, finalizer)
			err = r.Update(ctx, t)
			if err != nil {
				r.Logger.Error(err, "unable to remove finalizer from secret", "name", t.Name)
			}
			metrics.ManagedSecretCount.Dec()
		}
		return err
	default:
		return fmt.Errorf("unsupported type: %T", T)
	}

}

func (r *ArgoRolloutConfigKeeperCommon) ignoreExtraneousOperation(ctx context.Context, T interface{}, latestVersion *goversion.Version) error {
	switch t := T.(type) {
	case *corev1.ConfigMap:
		configMapVersion, err := goversion.NewVersion(t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			r.Logger.Error(err, "unable to parse version")
			return err
		}
		if configMapVersion.LessThan(latestVersion) {
			r.Logger.Info(fmt.Sprintf("adding IgnoreExtraneous annotation to %s configmap, reason: version is less than latest version", t.Name))
			if t.Annotations != nil {
				t.Annotations["argocd.argoproj.io/compare-options"] = "IgnoreExtraneous"
			} else {
				t.Annotations = map[string]string{"argocd.argoproj.io/compare-options": "IgnoreExtraneous"}
			}

			err = r.Update(ctx, t)
			if err != nil {
				r.Logger.Error(err, "unable to update configmap")
				return err
			}
		}
		return nil
	case *corev1.Secret:
		secretVersion, err := goversion.NewVersion(t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			r.Logger.Error(err, "unable to parse version")
			return err
		}
		if secretVersion.LessThan(latestVersion) {
			r.Logger.Info(fmt.Sprintf("adding IgnoreExtraneous annotation to %s secret, reason: version is less than latest version", t.Name))
			if t.Annotations != nil {
				t.Annotations["argocd.argoproj.io/compare-options"] = "IgnoreExtraneous"
			} else {
				t.Annotations = map[string]string{"argocd.argoproj.io/compare-options": "IgnoreExtraneous"}
			}
			err = r.Update(ctx, t)
			if err != nil {
				r.Logger.Error(err, "unable to update secret")
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", T)
	}
}

func (r *ArgoRolloutConfigKeeperCommon) getFilteredReplicaSets(ctx context.Context, namespace string, labelSelector map[string]string) (*appsv1.ReplicaSetList, error) {
	// need to list all ReplicaSets in namespace and filter by label
	replicaSets := &appsv1.ReplicaSetList{}

	if err := r.List(ctx, replicaSets, client.InNamespace(namespace), client.MatchingLabels(labelSelector)); err != nil {
		r.Logger.Error(err, fmt.Sprintf("unable to list replicasets in namespace %s", namespace))
		return nil, client.IgnoreNotFound(err)
	}

	return replicaSets, nil
}

func (r *ArgoRolloutConfigKeeperCommon) checkIfFinalizerInUse(ctx context.Context, namespace, appLabelValue, chartVersion string) (bool, error) {
	labelSelector := map[string]string{
		r.Labels.AppLabel:        appLabelValue,
		r.Labels.AppVersionLabel: chartVersion,
	}
	replicaSets, err := r.getFilteredReplicaSets(ctx, namespace, labelSelector)

	if err != nil {
		r.Logger.Error(err, "unable to get filtered replicasets")
		return false, err
	}

	for _, replicaSet := range replicaSets.Items {
		replicaNum := int32(0)

		if replicaSet.Labels[r.Labels.AppLabel] == appLabelValue && replicaSet.Labels[r.Labels.AppVersionLabel] == chartVersion && (*replicaSet.Spec.Replicas != replicaNum || replicaSet.Status.Replicas != replicaNum) {
			r.Logger.Info(fmt.Sprintf("finalizer in use by %s replicaset", replicaSet.Name))
			return true, nil
		}
	}
	return false, nil
}

func (r *ArgoRolloutConfigKeeperCommon) getLatestVersionOfReplicaSet(ctx context.Context, namespace, appLabelValue string) (*goversion.Version, error) {
	// need to list all ReplicaSets in namespace and filter by label
	labelSelector := map[string]string{
		r.Labels.AppLabel: appLabelValue,
	}
	replicaSets, err := r.getFilteredReplicaSets(ctx, namespace, labelSelector)

	if err != nil {
		r.Logger.Error(err, "unable to get filtered replicasets")
		return nil, err
	}

	var latestVersion *goversion.Version

	for _, replicaSet := range replicaSets.Items {

		if val, ok := replicaSet.Labels[r.Labels.AppVersionLabel]; ok {
			ver, err := goversion.NewVersion(val)
			if err != nil {
				r.Logger.Error(err, "unable to parse version")
				continue
			}
			if latestVersion == nil || ver.GreaterThan(latestVersion) {
				latestVersion = ver
			}
		} else {
			r.Logger.Info(fmt.Sprintf("replicaset %s does not have %s label", replicaSet.Name, r.Labels.AppVersionLabel))
			continue
		}
	}
	return latestVersion, nil
}

func (r *ArgoRolloutConfigKeeperCommon) UpdateStatus(ctx context.Context, T interface{}, status string) error {
	switch t := T.(type) {
	case *keeperv1alpha1.ArgoRolloutConfigKeeper:
		if t.Status.State != status {
			r.Logger.Info(fmt.Sprintf("updating %s ArgoRolloutConfigKeeper status from %s to %s", t.Name, t.Status.State, status))
			t.Status.State = status
			err := r.Status().Update(ctx, t)
			if err != nil {
				r.Logger.Error(err, "unable to update status")
				return err
			}
			// the sleep is to allow the status to be updated before the next reconcile
			time.Sleep(1 * time.Second)
		}
		return nil
	case *keeperv1alpha1.ArgoRolloutConfigKeeperClusterScope:
		if t.Status.State != status {
			r.Logger.Info(fmt.Sprintf("updating %s ArgoRolloutConfigKeeper status from %s to %s", t.Name, t.Status.State, status))
			t.Status.State = status
			err := r.Status().Update(ctx, t)
			if err != nil {
				r.Logger.Error(err, "unable to update status")
				return err
			}
			// the sleep is to allow the status to be updated before the next reconcile
			time.Sleep(1 * time.Second)
		}
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", T)
	}
}

func (r *ArgoRolloutConfigKeeperCommon) listConfigMaps(ctx context.Context, namespace string, labelSelector map[string]string) (*corev1.ConfigMapList, error) {
	configmaps := &corev1.ConfigMapList{}

	if err := r.List(ctx, configmaps, client.InNamespace(namespace), client.MatchingLabels(labelSelector)); err != nil {
		if namespace != "" {
			r.Logger.Error(err, fmt.Sprintf("unable to list configmaps in %s namespace", namespace))
		} else {
			r.Logger.Error(err, "unable to list configmaps")
		}
		return nil, client.IgnoreNotFound(err)
	}

	return configmaps, nil
}

func (r *ArgoRolloutConfigKeeperCommon) listSecrets(ctx context.Context, namespace string, labelSelector map[string]string) (*corev1.SecretList, error) {
	secrets := &corev1.SecretList{}

	if err := r.List(ctx, secrets, client.InNamespace(namespace), client.MatchingLabels(labelSelector)); err != nil {
		if namespace != "" {
			r.Logger.Error(err, fmt.Sprintf("unable to list secrets in %s namespace", namespace))
		} else {
			r.Logger.Error(err, "unable to list secrets")
		}
		return nil, client.IgnoreNotFound(err)
	}

	return secrets, nil
}

func (r *ArgoRolloutConfigKeeperCommon) getLatestVersionOfConfig(ctx context.Context, finalizerFullName string, T interface{}) (*goversion.Version, error) {

	switch t := T.(type) {
	case *corev1.ConfigMap:

		labels := tools.CopyMap(t.Labels)
		// need to remove the appVersion label from the labels to all configMaps
		delete(labels, r.Labels.AppVersionLabel)
		latestVersion, err := r.getLatestVersionOfReplicaSet(ctx, t.Namespace, finalizerFullName)
		if err != nil {
			r.Logger.Error(err, "unable to get latest version of replicaset")
			return nil, err
		}

		if latestVersion == nil {
			if configMaps, err := r.listConfigMaps(ctx, t.Namespace, labels); err != nil {
				return nil, err
			} else {
				for _, c := range configMaps.Items {
					if val, ok := c.Labels[r.Labels.AppVersionLabel]; ok {
						ver, err := goversion.NewVersion(val)
						if err != nil {
							r.Logger.Error(err, "unable to parse version")
							return nil, err
						}
						if latestVersion == nil || ver.GreaterThan(latestVersion) {
							latestVersion = ver
						}
					}
				}
			}
		}
		return latestVersion, nil
	case *corev1.Secret:
		labels := tools.CopyMap(t.Labels)

		// need to remove the appVersion label from the labels to all secrets
		delete(labels, r.Labels.AppVersionLabel)
		latestVersion, err := r.getLatestVersionOfReplicaSet(ctx, t.Namespace, finalizerFullName)
		if err != nil {
			r.Logger.Error(err, "unable to get latest version of replicaset")
			return nil, err
		}

		if latestVersion == nil {
			r.Logger.Info("could not get latest version from replicaset, trying to get from secrets")
			if secrets, err := r.listSecrets(ctx, t.Namespace, labels); err != nil {
				return nil, err
			} else {
				for _, c := range secrets.Items {
					if val, ok := c.Labels[r.Labels.AppVersionLabel]; ok {
						ver, err := goversion.NewVersion(val)
						if err != nil {
							r.Logger.Error(err, "unable to parse version")
							return nil, err
						}
						if latestVersion == nil || ver.GreaterThan(latestVersion) {
							latestVersion = ver
						}
					}
				}
			}
		}
		return latestVersion, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", T)
	}
}
