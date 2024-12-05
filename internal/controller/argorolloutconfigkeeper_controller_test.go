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
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	keeperv1alpha1 "github.com/run-ai/argo-rollout-config-keeper/api/v1alpha1"
	"github.com/run-ai/argo-rollout-config-keeper/internal/tools"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	sigclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ctx                      = context.Background()
	name                     = "argo-rollout-config-keeper"
	namespace                = "default"
	configMapNamespace       = "test-configmap"
	secretNamespace          = "test-secret"
	chartName                = "testing-chart"
	appVersion               = "1.24"
	appPreviewVersion        = "1.25"
	applicationName          = "test-application"
	finalizerName            = "argorolloutconfigkeeper.test"
	partOf                   = "app-testing"
	blueGreenPrefix          = "blue-green"
	blueGreenConfigMapPrefix = fmt.Sprintf("%s-configmap", blueGreenPrefix)
	blueGreenSecretPrefix    = fmt.Sprintf("%s-secret", blueGreenPrefix)
)

var _ = Describe("ArgoRolloutConfigKeeper Controller", Ordered, func() {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	argoRolloutConfigKeeper := &keeperv1alpha1.ArgoRolloutConfigKeeper{}

	Context("Blue-Green ConfigMap Tests", func() {
		BeforeAll(func() {
			// Install the CRDs
			manageRollouts(ctx, namespace, blueGreenPrefix, "create", blueGreenConfigMapPrefix)
			labelSelector := map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenConfigMapPrefix),
				"app.kubernetes.io/version": appVersion,
			}
			waitForReplicaSetToBeReady(ctx, namespace, labelSelector)
			manageConfigmaps(ctx, namespace, "create", blueGreenConfigMapPrefix)
			By("creating the custom resource for the Kind ArgoRolloutConfigKeeper")
			err := k8sClient.Get(ctx, typeNamespacedName, argoRolloutConfigKeeper)
			if err != nil && errors.IsNotFound(err) {
				resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: keeperv1alpha1.ArgoRolloutConfigKeeperSpec{
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
			manageConfigmaps(ctx, namespace, "delete", blueGreenConfigMapPrefix)
			manageRollouts(ctx, namespace, blueGreenPrefix, "delete", blueGreenConfigMapPrefix)
			resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeper")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the argo-rollout-config-keeper resource", func() {
			By("Reconciling the created argo-rollout-config-keeper resource")
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
		It("Blue-Green ConfigMap should have the correct annotations", func() {
			createNewRolloutRevision(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenConfigMapPrefix), blueGreenConfigMapPrefix)
			labelSelector := map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenConfigMapPrefix),
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

			configMap, err := getConfigmap(ctx, namespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenConfigMapPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
		It("Blue-Green ConfigMap should not have the finalizer", func() {
			promoteRollout(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenConfigMapPrefix))

			checkOldReplicaSet(ctx, namespace, map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenConfigMapPrefix),
				"app.kubernetes.io/version": appVersion,
			})

			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			configMap, err := getConfigmap(ctx, namespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenConfigMapPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Finalizers).To(BeNil())
		})
	})

	Context("Blue-Green Secret Tests", func() {
		BeforeAll(func() {
			// Install the CRDs
			manageRollouts(ctx, namespace, blueGreenPrefix, "create", blueGreenSecretPrefix)
			labelSelector := map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenSecretPrefix),
				"app.kubernetes.io/version": appVersion,
			}
			waitForReplicaSetToBeReady(ctx, namespace, labelSelector)
			manageSecrets(ctx, namespace, "create", blueGreenSecretPrefix)
			By("creating the custom resource for the Kind ArgoRolloutConfigKeeper")
			err := k8sClient.Get(ctx, typeNamespacedName, argoRolloutConfigKeeper)
			if err != nil && errors.IsNotFound(err) {
				resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: keeperv1alpha1.ArgoRolloutConfigKeeperSpec{
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
			manageSecrets(ctx, namespace, "delete", blueGreenSecretPrefix)
			manageRollouts(ctx, namespace, blueGreenPrefix, "delete", blueGreenSecretPrefix)
			resource := &keeperv1alpha1.ArgoRolloutConfigKeeper{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance ArgoRolloutConfigKeeper")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the argo-rollout-config-keeper resource", func() {
			By("Reconciling the created argo-rollout-config-keeper resource")
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
		It("Blue-Green Secret should have the correct annotations", func() {
			createNewRolloutRevision(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenSecretPrefix), blueGreenSecretPrefix)
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

			secret, err := getSecret(ctx, namespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenSecretPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Annotations).To(HaveKeyWithValue("argocd.argoproj.io/compare-options", "IgnoreExtraneous"))
		})
		It("Blue-Green Secret should not have the finalizer", func() {
			promoteRollout(ctx, namespace, fmt.Sprintf("%s-%s-%s", chartName, applicationName, blueGreenSecretPrefix))

			checkOldReplicaSet(ctx, namespace, map[string]string{
				"app.kubernetes.io/name":    fmt.Sprintf("%s-%s", applicationName, blueGreenSecretPrefix),
				"app.kubernetes.io/version": appVersion,
			})

			By("Reconciling the created resource")
			controllerReconciler := &ArgoRolloutConfigKeeperReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret, err := getSecret(ctx, namespace, fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, blueGreenSecretPrefix))
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Finalizers).To(BeNil())
		})
	})

})

func manageConfigmaps(ctx context.Context, namespace, operation, prefix string) {
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, prefix),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/part-of":    partOf,
				"app.kubernetes.io/version":    appVersion,
				"helm.sh/chart":                fmt.Sprintf("%s-%s", chartName, appVersion),
			},
			Finalizers: []string{fmt.Sprintf("%s/%s-%s", finalizerName, applicationName, prefix)},
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

func manageSecrets(ctx context.Context, namespace, operation, prefix string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s-%s", chartName, applicationName, appVersion, prefix),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/part-of":    partOf,
				"app.kubernetes.io/version":    appVersion,
				"helm.sh/chart":                fmt.Sprintf("%s-%s", chartName, appVersion),
			},
			Finalizers: []string{fmt.Sprintf("%s/%s-%s", finalizerName, applicationName, prefix)},
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

func manageRollouts(ctx context.Context, namespace, rolloutType, operation, prefix string) {
	activeServiceName := fmt.Sprintf("active-service-%s", prefix)
	previewServiceName := fmt.Sprintf("preview-service-%s", prefix)
	appName := fmt.Sprintf("%s-%s", applicationName, prefix)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      activeServiceName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/name":       appName,
				"app.kubernetes.io/part-of":    partOf,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/name":       appName,
				"app.kubernetes.io/part-of":    partOf,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}

	servicePreview := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      previewServiceName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/name":       appName,
				"app.kubernetes.io/part-of":    partOf,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/name":       appName,
				"app.kubernetes.io/part-of":    partOf,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}

	rollout := &v1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", chartName, applicationName, prefix),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": appName,
			},
		},
		Spec: v1alpha1.RolloutSpec{
			Replicas: tools.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
					"app.kubernetes.io/managed-by": "Helm",
					"app.kubernetes.io/name":       appName,
					"app.kubernetes.io/part-of":    partOf,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
						"app.kubernetes.io/managed-by": "Helm",
						"app.kubernetes.io/name":       appName,
						"app.kubernetes.io/part-of":    partOf,
						"app.kubernetes.io/version":    appVersion,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: fmt.Sprintf("nginx:%s", appVersion),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	if rolloutType == blueGreenPrefix {
		rollout.Spec.Strategy = v1alpha1.RolloutStrategy{
			BlueGreen: &v1alpha1.BlueGreenStrategy{
				AutoPromotionEnabled: tools.BoolPtr(false),
				ActiveService:        activeServiceName,
				ActiveMetadata: &v1alpha1.PodTemplateMetadata{
					Labels: map[string]string{
						"role": "active",
					},
				},
				PreviewService: previewServiceName,
				PreviewMetadata: &v1alpha1.PodTemplateMetadata{
					Labels: map[string]string{
						"role": "preview",
					},
				},
			},
		}
	}

	switch operation {
	case "create":
		// create service
		Expect(k8sClient.Create(ctx, service)).To(Succeed())
		Expect(k8sClient.Create(ctx, servicePreview)).To(Succeed())
		// create rollout
		Expect(k8sClient.Create(ctx, rollout)).To(Succeed())
		break
	case "delete":
		// delete service
		Expect(k8sClient.Delete(ctx, service)).To(Succeed())
		Expect(k8sClient.Delete(ctx, servicePreview)).To(Succeed())
		// delete rollout
		Expect(k8sClient.Delete(ctx, rollout)).To(Succeed())
		break
	default:
		panic("Invalid operation")
	}
}

func createNewRolloutRevision(ctx context.Context, namespace, rolloutName, prefix string) {
	//	need to create a new rollout revision
	rollout := &v1alpha1.Rollout{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: rolloutName, Namespace: namespace}, rollout)).To(Succeed())

	rollout.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s-%s", chartName, applicationName),
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/name":       fmt.Sprintf("%s-%s", applicationName, prefix),
				"app.kubernetes.io/part-of":    partOf,
				"app.kubernetes.io/version":    appPreviewVersion,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: fmt.Sprintf("nginx:%s", appPreviewVersion),
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}
	Expect(k8sClient.Update(ctx, rollout)).To(Succeed())
}

func promoteRollout(ctx context.Context, namespace, rolloutName string) {
	rollout := &v1alpha1.Rollout{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: rolloutName, Namespace: namespace}, rollout)).To(Succeed())

	rollout.Status.PromoteFull = true
	Expect(k8sClient.Status().Update(ctx, rollout)).To(Succeed())

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: rolloutName, Namespace: namespace}, rollout)).To(Succeed())
		return rollout.Status.Phase == "Healthy"
	}, "10s", "1s").Should(BeTrue())
}

func waitForReplicaSetToBeReady(ctx context.Context, namespace string, labelSelector map[string]string) {
	replicaSets := &appsv1.ReplicaSetList{}

	Eventually(func() bool {
		Expect(k8sClient.List(ctx, replicaSets, sigclient.InNamespace(namespace), sigclient.MatchingLabels(labelSelector))).To(Succeed())
		if replicaSets.Items == nil {
			return false
		}
		replicaSet := replicaSets.Items[0]
		return replicaSet.Status.ReadyReplicas == *replicaSet.Spec.Replicas
	}, "50s", "40s", "30s", "20s", "10s", "1s").Should(BeTrue())
}

// need to check the old replica set desired replicas is 0
func checkOldReplicaSet(ctx context.Context, namespace string, labelSelector map[string]string) {
	replicaSets := &appsv1.ReplicaSetList{}

	Eventually(func() bool {
		Expect(k8sClient.List(ctx, replicaSets, sigclient.InNamespace(namespace), sigclient.MatchingLabels(labelSelector))).To(Succeed())
		if replicaSets.Items == nil {
			return false
		}
		replicaSet := replicaSets.Items[0]
		return replicaSet.Status.ReadyReplicas == 0
	}, "50s", "40s", "30s", "20s", "10s", "1s").Should(BeTrue())
}
