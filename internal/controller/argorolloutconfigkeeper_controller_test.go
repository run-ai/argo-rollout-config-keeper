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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	keeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ctx                   = context.Background()
	name                  = "argo-rollout-config-keeper"
	namespace             = "default"
	configMapNamespace    = "test-configmap"
	secretNamespace       = "test-secret"
	chartName             = "testing-chart"
	appVersion            = "1.11.0-staging1"
	appPreviewVersion     = "1.11.0-staging2"
	applicationName       = "test-application"
	finalizerName         = "argorolloutconfigkeeper.test"
	partOf                = "app-testing"
	finalizerNameFullName = fmt.Sprintf("%s/%s", finalizerName, applicationName)
)

var _ = Describe("ArgoRolloutConfigKeeper Controller", Ordered, func() {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	argoRolloutConfigKeeper := &keeperv1alpha1.ArgoRolloutConfigKeeper{}

	Context("Configmap Tests", func() {
		BeforeAll(func() {
			manageConfigmaps(ctx, namespace, "create")
			manageReplicas(ctx, namespace, "create", 1)
			By("creating the custom resource for the Kind ArgoRolloutConfigKeeper")
			err := k8sClient.Get(ctx, typeNamespacedName, argoRolloutConfigKeeper)
			if err != nil && errors.IsNotFound(err) {
				resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: keeperv1alpha1.ArgoRolloutConfigKeeperSpec{
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
			manageConfigmaps(ctx, namespace, "delete")
			manageReplicas(ctx, namespace, "delete", 0)
			resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeper")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
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
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			configMap, err := getConfigmap(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Finalizers).To(ContainElement(finalizerNameFullName))
		})
		It("Should remove the finalizer from the configmap and add IgnoreExtraneous annotation", func() {
			By("Updating the replicaset to 0")
			manageReplicas(ctx, namespace, "update", 0)
			By("Reconciling the created resource")
			var (
				err       error
				configMap *corev1.ConfigMap
			)
			configMap, err = getConfigmap(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))

			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			configMap, err = getConfigmap(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Finalizers).To(BeNil())
			Expect(configMap.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
	})

	Context("Secret Tests", func() {
		BeforeAll(func() {
			manageSecrets(ctx, namespace, "create")
			manageReplicas(ctx, namespace, "create", 1)
			By("creating the custom resource for the Kind ArgoRolloutConfigKeeper")
			err := k8sClient.Get(ctx, typeNamespacedName, argoRolloutConfigKeeper)
			if err != nil && errors.IsNotFound(err) {
				resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: keeperv1alpha1.ArgoRolloutConfigKeeperSpec{
						// Add spec details here
						FinalizerName: finalizerName,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterAll(func() {
			manageSecrets(ctx, namespace, "delete")
			manageReplicas(ctx, namespace, "delete", 0)
			resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeper")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
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
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret, err := getSecret(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Finalizers).To(ContainElement(finalizerNameFullName))
		})
		It("Should remove the finalizer from the secret and add IgnoreExtraneous annotation", func() {
			By("Updating the replicaset to 0")
			manageReplicas(ctx, namespace, "update", 0)
			By("Reconciling the created resource")
			var (
				err    error
				secret *corev1.Secret
			)

			secret, err = getSecret(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))

			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret, err = getSecret(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Finalizers).To(BeNil())
			Expect(secret.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
	})

})

func manageConfigmaps(ctx context.Context, namespace, operation string) {

	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/part-of":    partOf,
				"app.kubernetes.io/version":    appVersion,
				"helm.sh/chart":                fmt.Sprintf("%s-%s", chartName, appVersion),
			},
			Finalizers: []string{finalizerNameFullName},
		},
	}

	switch operation {
	case "create":
		// create configmap
		Expect(k8sClient.Create(ctx, configmap)).To(Succeed())
		break
	case "delete":
		// delete configmap
		Expect(k8sClient.Delete(ctx, configmap)).To(Succeed())
		break
	case "update":
		// update configmap
		Expect(k8sClient.Update(ctx, configmap)).To(Succeed())
	default:
		panic("Invalid operation")
	}
}

func getConfigmap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, configmap)
	return configmap, err
}

func manageSecrets(ctx context.Context, namespace, operation string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", chartName, applicationName, appVersion),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/part-of":    partOf,
				"app.kubernetes.io/version":    appVersion,
				"helm.sh/chart":                fmt.Sprintf("%s-%s", chartName, appVersion),
			},
			Finalizers: []string{finalizerNameFullName},
		},
	}

	switch operation {
	case "create":
		// create secret
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		break
	case "delete":
		// delete secret
		Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		break
	case "update":
		// update secret
		Expect(k8sClient.Update(ctx, secret)).To(Succeed())
		break
	default:
		panic("Invalid operation")
	}
}

func getSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	return secret, err
}

func manageReplicas(ctx context.Context, namespace, operation string, replicaNumber int) {
	replicaNum := int32(replicaNumber)
	replica := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", chartName, applicationName, "db4d5cb8"),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/name":       applicationName,
				"app.kubernetes.io/part-of":    partOf,
				"app.kubernetes.io/version":    appVersion,
				"helm.sh/chart":                fmt.Sprintf("%s-%s", chartName, appVersion),
				"rollouts-pod-template-hash":   "db4d5cb8",
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicaNum,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
					"app.kubernetes.io/managed-by": "Helm",
					"app.kubernetes.io/name":       applicationName,
					"app.kubernetes.io/part-of":    partOf,
					"rollouts-pod-template-hash":   "db4d5cb8",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
						"app.kubernetes.io/managed-by": "Helm",
						"app.kubernetes.io/name":       applicationName,
						"app.kubernetes.io/part-of":    partOf,
						"rollouts-pod-template-hash":   "db4d5cb8",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "runai",
							Image: "alpine:3.14",
						},
					},
				},
			},
		},
	}

	replicaPreview := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", chartName, applicationName, "ab4d5c47"),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/name":       applicationName,
				"app.kubernetes.io/part-of":    partOf,
				"app.kubernetes.io/version":    appPreviewVersion,
				"helm.sh/chart":                fmt.Sprintf("%s-%s", chartName, appPreviewVersion),
				"rollouts-pod-template-hash":   "ab4d5c47",
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicaNum,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
					"app.kubernetes.io/managed-by": "Helm",
					"app.kubernetes.io/name":       applicationName,
					"app.kubernetes.io/part-of":    partOf,
					"rollouts-pod-template-hash":   "ab4d5c47",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
						"app.kubernetes.io/managed-by": "Helm",
						"app.kubernetes.io/name":       applicationName,
						"app.kubernetes.io/part-of":    partOf,
						"rollouts-pod-template-hash":   "ab4d5c47",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "runai",
							Image: "alpine:3.14",
						},
					},
				},
			},
		},
	}

	switch operation {
	case "create":
		// create replica
		Expect(k8sClient.Create(ctx, replica)).To(Succeed())
		Expect(k8sClient.Create(ctx, replicaPreview)).To(Succeed())
		time.Sleep(2 * time.Second)
		break
	case "delete":
		// delete replica
		Expect(k8sClient.Delete(ctx, replica)).To(Succeed())
		Expect(k8sClient.Delete(ctx, replicaPreview)).To(Succeed())
		time.Sleep(2 * time.Second)
		break
	case "update":
		// update replica
		Expect(k8sClient.Update(ctx, replica)).To(Succeed())
		Expect(k8sClient.Update(ctx, replicaPreview)).To(Succeed())
		time.Sleep(2 * time.Second)
		break
	default:
		panic("Invalid operation")
	}
}

func manageNamespace(ctx context.Context, name, operation string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	switch operation {
	case "create":
		// create namespace
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
		break
	case "delete":
		// delete namespace
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		break
	default:
		panic("Invalid operation")
	}
}
