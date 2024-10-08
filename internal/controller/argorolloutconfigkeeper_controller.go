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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	keeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
	"github.com/run-ai/argo-rollout-config-keeper/internal/common"
	"github.com/run-ai/argo-rollout-config-keeper/internal/metrics"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ArgoRolloutConfigKeeperReconciler reconciles a ArgoRolloutConfigKeeper object
type ArgoRolloutConfigKeeperReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

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
	configKeeperCommon := common.ArgoRolloutConfigKeeperCommon{
		Client: r.Client,
		Logger: r.logger,
	}

	configKeeper := &keeperv1alpha1.ArgoRolloutConfigKeeper{}
	if err := r.Get(ctx, req.NamespacedName, configKeeper); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	argoRolloutConfigStateInitializing := metav1.Condition{
		Type:    ArgoRolloutConfigStateInitializing,
		Status:  metav1.ConditionFalse,
		Reason:  "ArgoRolloutConfigKeeperInitializing",
		Message: "ArgoRolloutConfigKeeper is initializing",
	}

	if err := configKeeperCommon.UpdateCondition(ctx, configKeeper, argoRolloutConfigStateInitializing); err != nil {
		return ctrl.Result{}, err
	}

	configKeeperCommon.Labels = &common.ArgoRolloutConfigKeeperLabels{
		AppLabel:        configKeeper.Spec.AppLabel,
		AppVersionLabel: configKeeper.Spec.AppVersionLabel,
	}
	configKeeperCommon.FinalizerName = configKeeper.Spec.FinalizerName

	argoRolloutConfigStateReady := metav1.Condition{
		Type:    ArgoRolloutConfigStateReady,
		Status:  metav1.ConditionTrue,
		Reason:  "ArgoRolloutConfigKeeperReady",
		Message: "ArgoRolloutConfigKeeper is ready",
	}

	if err := configKeeperCommon.UpdateCondition(ctx, configKeeper, argoRolloutConfigStateReady); err != nil {
		return ctrl.Result{}, err
	}

	labelSelector := map[string]string{}
	if configKeeper.Spec.ConfigLabelSelector != nil {
		labelSelector = configKeeper.Spec.ConfigLabelSelector
	}

	if err := configKeeperCommon.ReconcileConfigMaps(ctx, req.Namespace, labelSelector, map[string]bool{}); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// need to list all secrets in namespace
	if err := configKeeperCommon.ReconcileSecrets(ctx, req.Namespace, labelSelector, map[string]bool{}); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
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
