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
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configkeeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
)

var _ = Describe("ArgoRolloutConfigKeeperClusterScope Controller", Ordered, func() {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	argoRolloutConfigKeeperClusterScope := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}

	Context("Blue-Green ConfigMap Tests", func() {
		BeforeAll(func() {
			manageNamespace(ctx, configMapNamespace, "create")
			manageRollouts(ctx, configMapNamespace, blueGreenPrefix, "create", blueGreenConfigMapPrefix)
			labelSelector := map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenConfigMapPrefix),
				"app.kubernetes.io/version": appVersion,
			}
			waitForReplicaSetToBeReady(ctx, configMapNamespace, labelSelector)
			manageConfigmaps(ctx, configMapNamespace, "create", blueGreenConfigMapPrefix)
			By("creating the custom resource for the Kind ArgoRolloutConfigKeeperClusterScope")
			err := k8sClient.Get(ctx, typeNamespacedName, argoRolloutConfigKeeperClusterScope)
			if err != nil && errors.IsNotFound(err) {
				resource := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScopeSpec{
						FinalizerName: finalizerName,
						ConfigLabelSelector: map[string]string{
							"app.kubernetes.io/part-of": partOf,
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterAll(func() {
			manageConfigmaps(ctx, configMapNamespace, "delete", blueGreenConfigMapPrefix)
			manageRollouts(ctx, configMapNamespace, blueGreenPrefix, "delete", blueGreenConfigMapPrefix)
			manageNamespace(ctx, configMapNamespace, "delete")
			resource := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeper")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the argo-rollout-config-keeper cluster resource", func() {
			By("Reconciling the created argo-rollout-config-keeper cluster resource")
			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
		It("Blue-Green ConfigMap should have the correct annotations", func() {
			createNewRolloutRevision(ctx, configMapNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenConfigMapPrefix), blueGreenConfigMapPrefix)
			labelSelector := map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenConfigMapPrefix),
				"app.kubernetes.io/version": appPreviewVersion,
			}
			waitForReplicaSetToBeReady(ctx, configMapNamespace, labelSelector)
			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())

			configMap, err := getConfigmap(ctx, configMapNamespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenConfigMapPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
		It("Blue-Green ConfigMap should not have the finalizer", func() {
			promoteRollout(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenConfigMapPrefix))

			checkOldReplicaSet(ctx, configMapNamespace, map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenConfigMapPrefix),
				"app.kubernetes.io/version": appVersion,
			})

			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			configMap, err := getConfigmap(ctx, configMapNamespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenConfigMapPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Finalizers).To(BeNil())
		})
	})

	Context("Blue-Green Secret Tests", func() {
		BeforeAll(func() {
			manageNamespace(ctx, secretNamespace, "create")
			manageRollouts(ctx, secretNamespace, blueGreenPrefix, "create", blueGreenSecretPrefix)
			labelSelector := map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenSecretPrefix),
				"app.kubernetes.io/version": appVersion,
			}
			waitForReplicaSetToBeReady(ctx, secretNamespace, labelSelector)
			manageSecrets(ctx, secretNamespace, "create", blueGreenSecretPrefix)
			By("creating the custom resource for the Kind ArgoRolloutConfigKeeperClusterScope")
			err := k8sClient.Get(ctx, typeNamespacedName, argoRolloutConfigKeeperClusterScope)
			if err != nil && errors.IsNotFound(err) {
				resource := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScopeSpec{
						// Add spec details here
						FinalizerName: finalizerName,
						IgnoredNamespaces: []string{
							"kube-system",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterAll(func() {
			manageRollouts(ctx, secretNamespace, blueGreenPrefix, "delete", blueGreenPrefix)
			manageSecrets(ctx, secretNamespace, "delete", blueGreenSecretPrefix)
			manageNamespace(ctx, secretNamespace, "delete")
			resource := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeperClusterScope")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the argo-rollout-config-keeper cluster scope resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
		It("Blue-Green Secret should have the correct annotations", func() {
			createNewRolloutRevision(ctx, secretNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenSecretPrefix), blueGreenSecretPrefix)
			labelSelector := map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenSecretPrefix),
				"app.kubernetes.io/version": appPreviewVersion,
			}
			waitForReplicaSetToBeReady(ctx, namespace, labelSelector)
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())

			secret, err := getSecret(ctx, secretNamespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenSecretPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
		It("Blue-Green Secret should not have the finalizer", func() {
			promoteRollout(ctx, secretNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenSecretPrefix))

			checkOldReplicaSet(ctx, secretNamespace, map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenSecretPrefix),
				"app.kubernetes.io/version": appVersion,
			})

			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret, err := getSecret(ctx, secretNamespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenSecretPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Finalizers).To(BeNil())
		})
	})

})
