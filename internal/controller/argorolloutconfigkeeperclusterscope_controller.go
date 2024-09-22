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

	"github.com/go-logr/logr"
	configkeeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
	"github.com/run-ai/argo-rollout-config-keeper/internal/common"
	"github.com/run-ai/argo-rollout-config-keeper/internal/metrics"
	"github.com/run-ai/argo-rollout-config-keeper/internal/tools"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ArgoRolloutConfigKeeperClusterScopeReconciler reconciles a ArgoRolloutConfigKeeperClusterScope object
type ArgoRolloutConfigKeeperClusterScopeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

//+kubebuilder:rbac:groups=configkeeper.run.ai,resources=argorolloutconfigkeeperclusterscopes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=configkeeper.run.ai,resources=argorolloutconfigkeeperclusterscopes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=configkeeper.run.ai,resources=argorolloutconfigkeeperclusterscopes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch

func (r *ArgoRolloutConfigKeeperClusterScopeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx)
	defer func() {
		metrics.OverallReconcileDuration.Observe(time.Since(time.Now()).Seconds())
	}()

	configKeeperCommon := common.ArgoRolloutConfigKeeperCommon{
		Client: r.Client,
		Scheme: r.Scheme,
		Logger: r.logger,
	}

	configKeeperClusterScope := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}
	if err := r.Get(ctx, req.NamespacedName, configKeeperClusterScope); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
	}

	if err := configKeeperCommon.UpdateStatus(ctx, configKeeperClusterScope, ArgoRolloutConfigStateInitializing); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	configKeeperCommon.Labels = &common.ArgoRolloutConfigKeeperLabels{
		AppLabel:        configKeeperClusterScope.Spec.AppLabel,
		AppVersionLabel: configKeeperClusterScope.Spec.AppVersionLabel,
	}
	configKeeperCommon.FinalizerName = configKeeperClusterScope.Spec.FinalizerName

	if err := configKeeperCommon.UpdateStatus(ctx, configKeeperClusterScope, ArgoRolloutConfigStateReconcilingConfigmaps); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	labelSelector := map[string]string{}
	if configKeeperClusterScope.Spec.ConfigLabelSelector != nil {
		labelSelector = configKeeperClusterScope.Spec.ConfigLabelSelector
	}

	ignoredNamespaces := tools.CreateMapFromStringList(configKeeperClusterScope.Spec.IgnoredNamespaces)

	if err := configKeeperCommon.ReconcileConfigMaps(ctx, "", labelSelector, ignoredNamespaces); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
	}

	if err := configKeeperCommon.UpdateStatus(ctx, configKeeperClusterScope, ArgoRolloutConfigStateReconcilingSecrets); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}
	// need to list all secrets in namespace
	if err := configKeeperCommon.ReconcileSecrets(ctx, "", labelSelector, ignoredNamespaces); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, client.IgnoreNotFound(err)
	}

	if err := configKeeperCommon.UpdateStatus(ctx, configKeeperClusterScope, ArgoRolloutConfigStateFinished); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoRolloutConfigKeeperClusterScopeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
