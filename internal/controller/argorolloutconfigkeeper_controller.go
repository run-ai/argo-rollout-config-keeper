/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	goversion "github.com/hashicorp/go-version"
	keeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
	"github.com/run-ai/argo-rollout-config-keeper/internal"
	"github.com/run-ai/argo-rollout-config-keeper/internal/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ArgoRolloutConfigKeeperReconciler reconciles a ArgoRolloutConfigKeeper object

type ArgoRolloutConfigKeeperLabels struct {
	AppLabel        string
	AppVersionLabel string
}

type ArgoRolloutConfigKeeperReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
	Labels *ArgoRolloutConfigKeeperLabels
}

const (
	ArgoRolloutConfigStateInitializing          = "initializing"
	ArgoRolloutConfigStateReconcilingConfigmaps = "reconciling configmaps"
	ArgoRolloutConfigStateReconcilingSecrets    = "reconciling secrets"
	ArgoRolloutConfigStateFinished              = "finished"
)

//+kubebuilder:rbac:groups=configkeeper.run.ai,resources=argorolloutconfigkeepers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=configkeeper.run.ai,resources=argorolloutconfigkeepers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=configkeeper.run.ai,resources=argorolloutconfigkeepers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch

func (r *ArgoRolloutConfigKeeperReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx)
	defer func() {
		metrics.OverallReconcileDuration.Observe(time.Since(time.Now()).Seconds())
	}()

	//r.logger.Error(nil, "reconciling ArgoRolloutConfigKeeper")
	configKeeper := &keeperv1alpha1.ArgoRolloutConfigKeeper{}
	if err := r.Get(ctx, req.NamespacedName, configKeeper); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
	}
	if err := r.updateStatus(ctx, configKeeper, ArgoRolloutConfigStateInitializing); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	r.Labels = &ArgoRolloutConfigKeeperLabels{
		AppLabel:        configKeeper.Spec.AppLabel,
		AppVersionLabel: configKeeper.Spec.AppVersionLabel,
	}
	if err := r.updateStatus(ctx, configKeeper, ArgoRolloutConfigStateReconcilingConfigmaps); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}
	configmaps := &corev1.ConfigMapList{}
	r.logger.Info(fmt.Sprintf("listing configmaps in %s namespace", req.Namespace))

	labelSelector := map[string]string{}
	if configKeeper.Spec.ConfigLabelSelector != nil {
		labelSelector = configKeeper.Spec.ConfigLabelSelector
	}

	if err := r.List(ctx, configmaps, client.InNamespace(req.Namespace), client.MatchingLabels(labelSelector)); err != nil {
		r.logger.Error(err, fmt.Sprintf("unable to list configmaps in %s namespace", req.Namespace))
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
	}
	metrics.DiscoveredConfigMapCount.Set(float64(len(configmaps.Items)))
	if configmaps.Items != nil {
		if err := r.reconcileConfigMaps(ctx, configmaps, configKeeper); err != nil {
			r.logger.Error(err, "unable to reconcile configmaps")
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
		}
	} else {
		r.logger.Info(fmt.Sprintf("no configmaps found in %s namespace", req.Namespace))
	}
	if err := r.updateStatus(ctx, configKeeper, ArgoRolloutConfigStateReconcilingSecrets); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}
	// need to list all secrets in namespace
	secrets := &corev1.SecretList{}
	if err := r.List(ctx, secrets, client.InNamespace(req.Namespace), client.MatchingLabels(labelSelector)); err != nil {
		r.logger.Error(err, fmt.Sprintf("unable to list secrets in %s namespace", req.Namespace))
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
	}
	metrics.DiscoveredSecretCount.Set(float64(len(secrets.Items)))
	if secrets.Items != nil {
		if err := r.reconcileSecrets(ctx, secrets, configKeeper); err != nil {
			r.logger.Error(err, "unable to reconcile secrets")
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
		}
	} else {
		r.logger.Info(fmt.Sprintf("no secrets found in %s namespace", req.Namespace))
	}
	if err := r.updateStatus(ctx, configKeeper, ArgoRolloutConfigStateFinished); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoRolloutConfigKeeperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keeperv1alpha1.ArgoRolloutConfigKeeper{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *ArgoRolloutConfigKeeperReconciler) reconcileConfigMaps(ctx context.Context, configmaps *corev1.ConfigMapList, configKeeper *keeperv1alpha1.ArgoRolloutConfigKeeper) error {
	defer func() {
		metrics.ConfigMapReconcileDuration.Observe(time.Since(time.Now()).Seconds())
	}()
	metrics.ManagedConfigMapCount.Set(0)
	for _, c := range configmaps.Items {
		r.logger.Info(fmt.Sprintf("configmap, name: %s", c.Name))
		if c.Finalizers != nil {
			// check if the finalizer is in the finalizers list
			finalizerFullName, havingManagedFinalizer := internal.ContainsString(c.GetFinalizers(), configKeeper.Spec.FinalizerName)

			if havingManagedFinalizer {
				metrics.ManagedConfigMapCount.Inc()
				if err := r.finalizerOperation(ctx, &c, finalizerFullName); err != nil {
					return err
				}

				if latestVersion, err := r.getLatestVersionOfReplicaSet(ctx, c.Namespace, strings.Split(finalizerFullName, "/")[1]); err != nil {
					r.logger.Error(err, "unable to get latest version of replicaset")
					return err
				} else {
					if err := r.ignoreExtraneousOperation(ctx, &c, latestVersion); err != nil {
						return err
					}
				}
			} else {
				r.logger.Info(fmt.Sprintf("skipping %s configmap, reason: no manageable finalizer", c.Name))
			}
			continue
		}
		r.logger.Info(fmt.Sprintf("skipping %s configmap, reason: no finalizers", c.Name))
	}
	return nil
}

func (r *ArgoRolloutConfigKeeperReconciler) reconcileSecrets(ctx context.Context, secrets *corev1.SecretList, configKeeper *keeperv1alpha1.ArgoRolloutConfigKeeper) error {
	defer func() {
		metrics.SecretReconcileDuration.Observe(time.Since(time.Now()).Seconds())
	}()
	metrics.ManagedSecretCount.Set(0)
	for _, s := range secrets.Items {
		r.logger.Info(fmt.Sprintf("secret, name: %s", s.Name))
		if s.Finalizers != nil {
			finalizerFullName, havingManagedFinalizer := internal.ContainsString(s.GetFinalizers(), configKeeper.Spec.FinalizerName)

			if havingManagedFinalizer {
				metrics.ManagedSecretCount.Inc()
				if err := r.finalizerOperation(ctx, &s, finalizerFullName); err != nil {
					return err
				}

				if latestVersion, err := r.getLatestVersionOfReplicaSet(ctx, s.Namespace, strings.Split(finalizerFullName, "/")[1]); err != nil {
					r.logger.Error(err, "unable to get latest version of replicaset")
					return err
				} else {
					if err := r.ignoreExtraneousOperation(ctx, &s, latestVersion); err != nil {
						return err
					}
				}
			} else {
				r.logger.Info(fmt.Sprintf("skipping %s secret, reason: no manageable finalizer", s.Name))
			}
			continue
		}
		r.logger.Info(fmt.Sprintf("skipping %s secret, reason: no finalizers", s.Name))
	}
	return nil
}

func (r *ArgoRolloutConfigKeeperReconciler) finalizerOperation(ctx context.Context, T interface{}, finalizer string) error {
	// Check the type of the object
	switch t := T.(type) {
	case *corev1.ConfigMap:
		r.logger.Info(fmt.Sprintf("finalizer operation on configmap object, name: %s", t.Name))
		inUse, err := r.checkIfFinalizerInUse(ctx, t.Namespace, strings.Split(finalizer, "/")[1], t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			return err
		}
		if !inUse {
			r.logger.Info(fmt.Sprintf("removing finalizer from configmap, name: %s, reason: finalizer not in use", t.Name))
			t.ObjectMeta.Finalizers = internal.RemoveString(t.ObjectMeta.Finalizers, finalizer)
			err = r.Update(ctx, t)
			if err != nil {
				r.logger.Error(err, "unable to remove finalizer from configmap", "name", t.Name)
			}
			metrics.ManagedConfigMapCount.Dec()
		}
		return err
	case *corev1.Secret:
		r.logger.Info(fmt.Sprintf("finalizer operation on secret object, name: %s", t.Name))
		inUse, err := r.checkIfFinalizerInUse(ctx, t.Namespace, strings.Split(finalizer, "/")[1], t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			return err
		}
		if !inUse {
			r.logger.Info(fmt.Sprintf("removing finalizer from secret, name: %s, reason: finalizer not in use", t.Name))
			t.ObjectMeta.Finalizers = internal.RemoveString(t.ObjectMeta.Finalizers, finalizer)
			err = r.Update(ctx, t)
			if err != nil {
				r.logger.Error(err, "unable to remove finalizer from secret", "name", t.Name)
			}
			metrics.ManagedSecretCount.Dec()
		}
		return err
	default:
		return fmt.Errorf("unsupported type: %T", T)
	}

}

func (r *ArgoRolloutConfigKeeperReconciler) ignoreExtraneousOperation(ctx context.Context, T interface{}, latestVersion *goversion.Version) error {
	switch t := T.(type) {
	case *corev1.ConfigMap:
		configMapVersion, err := goversion.NewVersion(t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			r.logger.Error(err, "unable to parse version")
			return err
		}
		if configMapVersion.LessThan(latestVersion) {
			r.logger.Info(fmt.Sprintf("adding IgnoreExtraneous annotation to %s configmap, reason: version is less than latest version", t.Name))
			if t.Annotations != nil {
				t.Annotations["argocd.argoproj.io/compare-options"] = "IgnoreExtraneous"
			} else {
				t.Annotations = map[string]string{"argocd.argoproj.io/compare-options": "IgnoreExtraneous"}
			}

			err = r.Update(ctx, t)
			if err != nil {
				r.logger.Error(err, "unable to update configmap")
				return err
			}
		}
		return nil
	case *corev1.Secret:
		secretVersion, err := goversion.NewVersion(t.Labels[r.Labels.AppVersionLabel])
		if err != nil {
			r.logger.Error(err, "unable to parse version")
			return err
		}
		if secretVersion.LessThan(latestVersion) {
			r.logger.Info(fmt.Sprintf("adding IgnoreExtraneous annotation to %s secret, reason: version is less than latest version", t.Name))
			if t.Annotations != nil {
				t.Annotations["argocd.argoproj.io/compare-options"] = "IgnoreExtraneous"
			} else {
				t.Annotations = map[string]string{"argocd.argoproj.io/compare-options": "IgnoreExtraneous"}
			}
			err = r.Update(ctx, t)
			if err != nil {
				r.logger.Error(err, "unable to update secret")
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", T)
	}
}

func (r *ArgoRolloutConfigKeeperReconciler) getFilteredReplicaSets(ctx context.Context, namespace string, labelSelector map[string]string) (*appsv1.ReplicaSetList, error) {
	// need to list all ReplicaSets in namespace and filter by label
	replicaSets := &appsv1.ReplicaSetList{}

	if err := r.List(ctx, replicaSets, client.InNamespace(namespace), client.MatchingLabels(labelSelector)); err != nil {
		r.logger.Error(err, fmt.Sprintf("unable to list replicasets in namespace %s", namespace))
		return nil, client.IgnoreNotFound(err)
	}

	return replicaSets, nil
}

func (r *ArgoRolloutConfigKeeperReconciler) checkIfFinalizerInUse(ctx context.Context, namespace, appLabelValue, chartVersion string) (bool, error) {
	labelSelector := map[string]string{
		r.Labels.AppLabel:        appLabelValue,
		r.Labels.AppVersionLabel: chartVersion,
	}
	replicaSets, err := r.getFilteredReplicaSets(ctx, namespace, labelSelector)

	if err != nil {
		r.logger.Error(err, "unable to get filtered replicasets")
		return false, err
	}

	for _, replicaSet := range replicaSets.Items {
		replicaNum := int32(0)

		if replicaSet.Labels[r.Labels.AppLabel] == appLabelValue && replicaSet.Labels[r.Labels.AppVersionLabel] == chartVersion && (*replicaSet.Spec.Replicas != replicaNum || replicaSet.Status.Replicas != replicaNum) {
			r.logger.Info(fmt.Sprintf("finalizer in use by %s replicaset", replicaSet.Name))
			return true, nil
		}
	}
	return false, nil
}

func (r *ArgoRolloutConfigKeeperReconciler) getLatestVersionOfReplicaSet(ctx context.Context, namespace, appLabelValue string) (*goversion.Version, error) {
	// need to list all ReplicaSets in namespace and filter by label
	labelSelector := map[string]string{
		r.Labels.AppLabel: appLabelValue,
	}
	replicaSets, err := r.getFilteredReplicaSets(ctx, namespace, labelSelector)

	if err != nil {
		r.logger.Error(err, "unable to get filtered replicasets")
		return nil, err
	}

	var latestVersion *goversion.Version

	for _, replicaSet := range replicaSets.Items {

		if val, ok := replicaSet.Labels[r.Labels.AppVersionLabel]; ok {
			ver, err := goversion.NewVersion(val)
			if err != nil {
				r.logger.Error(err, "unable to parse version")
				continue
			}
			if latestVersion == nil || ver.GreaterThan(latestVersion) {
				latestVersion = ver
			}
		} else {
			r.logger.Info(fmt.Sprintf("replicaset %s does not have %s label", replicaSet.Name, r.Labels.AppVersionLabel))
			continue
		}
	}
	return latestVersion, nil
}

func (r *ArgoRolloutConfigKeeperReconciler) updateStatus(ctx context.Context, configKeeper *keeperv1alpha1.ArgoRolloutConfigKeeper, status string) error {
	if configKeeper.Status.State != status {
		r.logger.Info(fmt.Sprintf("updating %s ArgoRolloutConfigKeeper status from %s to %s", configKeeper.Name, configKeeper.Status.State, status))
		configKeeper.Status.State = status
		err := r.Status().Update(ctx, configKeeper)
		if err != nil {
			r.logger.Error(err, "unable to update status")
			return err
		}
		// the sleep is to allow the status to be updated before the next reconcile
		time.Sleep(1 * time.Second)
	}
	return nil
}
