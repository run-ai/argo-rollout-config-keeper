package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/run-ai/argo-rollout-config-keeper/internal/tools"

	"github.com/go-logr/logr"
	keeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
	"github.com/run-ai/argo-rollout-config-keeper/internal/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ArgoRolloutConfigKeeperLabels struct {
	AppLabel        string
	AppVersionLabel string
}

type ArgoRolloutConfigKeeperCommon struct {
	client.Client
	Logger        logr.Logger
	Labels        *ArgoRolloutConfigKeeperLabels
	FinalizerName string
}

func (r *ArgoRolloutConfigKeeperCommon) ReconcileConfigMaps(ctx context.Context, namespace string, labelSelector map[string]string, ignoredNamespaces map[string]bool) error {
	defer func() {
		if namespace != "" {
			metrics.ConfigMapReconcileDuration.Observe(time.Since(time.Now()).Seconds())
		} else {
			metrics.ConfigMapClusterScopeReconcileDuration.Observe(time.Since(time.Now()).Seconds())
		}
	}()

	if namespace != "" {
		metrics.ManagedConfigMapCount.Set(0)
	} else {
		metrics.ManagedConfigMapClusterScopeCount.Set(0)
	}

	configMaps, err := r.listConfigMaps(ctx, namespace, labelSelector)
	if err != nil {
		return err
	}
	if namespace != "" {
		metrics.DiscoveredConfigMapCount.Set(float64(len(configMaps.Items)))
	} else {
		metrics.DiscoveredConfigMapClusterScopeCount.Set(float64(len(configMaps.Items)))
	}

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
					if namespace != "" {
						metrics.ManagedConfigMapCount.Inc()
					} else {
						metrics.ManagedConfigMapClusterScopeCount.Inc()
					}

					if err := r.finalizerOperation(ctx, &c, finalizerFullName); err != nil {
						r.Logger.Error(err, "unable to remove finalizer from configmap")
						if namespace != "" {
							metrics.FailuresInConfigMapClusterScopeCount.Inc()
						} else {
							metrics.FailuresInConfigMapCount.Inc()
						}
						continue
					}

					err = r.ignoreExtraneousOperation(ctx, &c, strings.Split(finalizerFullName, "/")[1])
					if err != nil {
						r.Logger.Error(err, "unable to add IgnoreExtraneous annotation to configmap")
						if namespace != "" {
							metrics.FailuresInConfigMapClusterScopeCount.Inc()
						} else {
							metrics.FailuresInConfigMapCount.Inc()
						}
						continue
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

	return nil
}

func (r *ArgoRolloutConfigKeeperCommon) ReconcileSecrets(ctx context.Context, namespace string, labelSelector map[string]string, ignoredNamespaces map[string]bool) error {
	defer func() {
		if namespace != "" {
			metrics.SecretReconcileDuration.Observe(time.Since(time.Now()).Seconds())
		} else {
			metrics.SecretClusterScopeReconcileDuration.Observe(time.Since(time.Now()).Seconds())
		}
	}()
	if namespace != "" {
		metrics.ManagedSecretCount.Set(0)
	} else {
		metrics.ManagedSecretClusterScopeCount.Set(0)
	}

	secrets, err := r.listSecrets(ctx, namespace, labelSelector)
	if err != nil {
		return err
	}
	if namespace != "" {
		metrics.DiscoveredSecretCount.Set(float64(len(secrets.Items)))
	} else {
		metrics.DiscoveredSecretClusterScopeCount.Set(float64(len(secrets.Items)))
	}

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
					if namespace != "" {
						metrics.ManagedSecretCount.Inc()
					} else {
						metrics.ManagedSecretClusterScopeCount.Inc()
					}

					if err := r.finalizerOperation(ctx, &s, finalizerFullName); err != nil {
						r.Logger.Error(err, "unable to remove finalizer from secret")
						if namespace != "" {
							metrics.FailuresInSecretClusterScopeCount.Inc()
						} else {
							metrics.FailuresInSecretCount.Inc()
						}
						return err
					}

					err = r.ignoreExtraneousOperation(ctx, &s, strings.Split(finalizerFullName, "/")[1])
					if err != nil {
						r.Logger.Error(err, "unable to add IgnoreExtraneous annotation to secret")
						if namespace != "" {
							metrics.FailuresInSecretClusterScopeCount.Inc()
						} else {
							metrics.FailuresInSecretCount.Inc()
						}
						continue
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
			if t.Namespace != "" {
				metrics.ManagedConfigMapClusterScopeCount.Dec()
			} else {
				metrics.ManagedConfigMapCount.Dec()
			}
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
			if t.Namespace != "" {
				metrics.ManagedSecretClusterScopeCount.Dec()
			} else {
				metrics.ManagedSecretCount.Dec()
			}
		}
		return err
	default:
		return fmt.Errorf("unsupported type: %T", T)
	}

}

func (r *ArgoRolloutConfigKeeperCommon) ignoreExtraneousOperation(ctx context.Context, T interface{}, appLabelValue string) error {
	switch t := T.(type) {
	case *corev1.ConfigMap:
		labelSelector := map[string]string{
			r.Labels.AppLabel:        appLabelValue,
			r.Labels.AppVersionLabel: t.Labels[r.Labels.AppVersionLabel],
		}
		replicaSets, err := r.getFilteredReplicaSets(ctx, t.Namespace, labelSelector)

		if err != nil {
			r.Logger.Error(err, "unable to get filtered replicasets")
			return err
		}

		rsAnnotation := replicaSets.Items[0].Annotations["rollout.argoproj.io/ephemeral-metadata"]
		// if replicaSet Annotation is not nil or empty, and having the following value: '{"labels":{"role":"preview"}}' or '{"labels":{"role":"canary"}}' it should add IgnoreExtraneous annotation
		if rsAnnotation != "" && (rsAnnotation == `{"labels":{"role":"preview"}}` || rsAnnotation == `{"labels":{"role":"canary"}}`) {
			r.Logger.Info(fmt.Sprintf("adding IgnoreExtraneous annotation to %s configmap, reason: replica has rollout.argoproj.io/ephemeral-metadata annotation", t.Name))
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
		labelSelector := map[string]string{
			r.Labels.AppLabel:        appLabelValue,
			r.Labels.AppVersionLabel: t.Labels[r.Labels.AppVersionLabel],
		}
		replicaSets, err := r.getFilteredReplicaSets(ctx, t.Namespace, labelSelector)

		if err != nil {
			r.Logger.Error(err, "unable to get filtered replicasets")
			return err
		}

		rsAnnotation := replicaSets.Items[0].Annotations["rollout.argoproj.io/ephemeral-metadata"]
		// if replicaSet Annotation is not nil or empty, and having the following value: '{"labels":{"role":"preview"}}' or '{"labels":{"role":"canary"}}' it should add IgnoreExtraneous annotation
		if rsAnnotation != "" && (rsAnnotation == `{"labels":{"role":"preview"}}` || rsAnnotation == `{"labels":{"role":"canary"}}`) {
			r.Logger.Info(fmt.Sprintf("adding IgnoreExtraneous annotation to %s secret, reason: replica has rollout.argoproj.io/ephemeral-metadata annotation", t.Name))
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

		if *replicaSet.Spec.Replicas != replicaNum || replicaSet.Status.Replicas != replicaNum {
			r.Logger.Info(fmt.Sprintf("finalizer in use by %s replicaset", replicaSet.Name))
			return true, nil
		}
	}
	return false, nil
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

func (r *ArgoRolloutConfigKeeperCommon) UpdateCondition(ctx context.Context, T interface{}, condition metav1.Condition) error {
	switch t := T.(type) {
	case *keeperv1alpha1.ArgoRolloutConfigKeeper:
		changed := meta.SetStatusCondition(&t.Status.Conditions, condition)
		if !changed {
			return nil
		}
		return r.Status().Update(ctx, t)
	case *keeperv1alpha1.ArgoRolloutConfigKeeperClusterScope:
		changed := meta.SetStatusCondition(&t.Status.Conditions, condition)
		if !changed {
			return nil
		}
		return r.Status().Update(ctx, t)
	default:
		return fmt.Errorf("unsupported type: %T", T)
	}
}
