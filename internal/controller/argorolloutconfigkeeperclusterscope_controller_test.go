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
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configkeeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
)

var _ = Describe("ArgoRolloutConfigKeeper Controller", Ordered, func() {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	argoRolloutConfigKeeperClusterScope := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}

	Context("Configmap Tests", func() {
		BeforeAll(func() {
			manageNamespace(ctx, configMapNamespace, "create")
			manageConfigmaps(ctx, configMapNamespace, "create")
			manageReplicas(ctx, configMapNamespace, "create", 1)
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
						ConfigLabelSelector: map[string]string{
							"app.kubernetes.io/part-of": partOf,
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterAll(func() {
			manageConfigmaps(ctx, configMapNamespace, "delete")
			manageReplicas(ctx, configMapNamespace, "delete", 0)
			manageNamespace(ctx, configMapNamespace, "delete")
			resource := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeper")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
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
		It("ConfigMap should have the finalizer", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			configMap, err := getConfigmap(ctx, configMapNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Finalizers).To(ContainElement(finalizerNameFullName))
		})
		It("Should remove the finalizer from the configmap and add IgnoreExtraneous annotation", func() {
			By("Updating the replicaset to 0")
			manageReplicas(ctx, configMapNamespace, "update", 0)
			By("Reconciling the created resource")
			var (
				err       error
				configMap *corev1.ConfigMap
			)
			configMap, err = getConfigmap(ctx, configMapNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))

			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			configMap, err = getConfigmap(ctx, configMapNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Finalizers).To(BeNil())
			Expect(configMap.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
	})

	Context("Secret Tests", func() {
		BeforeAll(func() {
			manageNamespace(ctx, secretNamespace, "create")
			manageSecrets(ctx, secretNamespace, "create")
			manageReplicas(ctx, secretNamespace, "create", 1)
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
			manageNamespace(ctx, secretNamespace, "delete")
			manageSecrets(ctx, secretNamespace, "delete")
			manageReplicas(ctx, secretNamespace, "delete", 0)
			resource := &configkeeperv1alpha1.ArgoRolloutConfigKeeperClusterScope{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeperClusterScope")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
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
		It("Secret should have the finalizer", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret, err := getSecret(ctx, secretNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Finalizers).To(ContainElement(finalizerNameFullName))
		})
		It("Should remove the finalizer from the secret and add IgnoreExtraneous annotation", func() {
			By("Updating the replicaset to 0")
			manageReplicas(ctx, secretNamespace, "update", 0)
			By("Reconciling the created resource")
			var (
				err    error
				secret *corev1.Secret
			)

			secret, err = getSecret(ctx, secretNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))

			controllerReconciler := &ArgoRolloutConfigKeeperClusterScopeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret, err = getSecret(ctx, secretNamespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Finalizers).To(BeNil())
			Expect(secret.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
	})
})
