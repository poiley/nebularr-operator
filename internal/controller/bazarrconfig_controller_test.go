/*
Copyright 2026.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
)

var _ = Describe("BazarrConfig Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When reconciling a BazarrConfig resource", func() {
		const (
			resourceName     = "test-bazarr"
			namespace        = "default"
			sonarrSecretName = "sonarr-credentials"
			radarrSecretName = "radarr-credentials"
		)

		var (
			ctx               context.Context
			typeNamespaceName types.NamespacedName
			bazarrConfig      *arrv1alpha1.BazarrConfig
			sonarrSecret      *corev1.Secret
			radarrSecret      *corev1.Secret
			reconciler        *BazarrConfigReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			typeNamespaceName = types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			// Create Sonarr credentials secret
			sonarrSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sonarrSecretName,
					Namespace: namespace,
				},
				StringData: map[string]string{
					"apiKey": "test-sonarr-api-key",
				},
			}
			err := k8sClient.Create(ctx, sonarrSecret)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			// Create Radarr credentials secret
			radarrSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      radarrSecretName,
					Namespace: namespace,
				},
				StringData: map[string]string{
					"apiKey": "test-radarr-api-key",
				},
			}
			err = k8sClient.Create(ctx, radarrSecret)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			// Create BazarrConfig resource
			bazarrConfig = &arrv1alpha1.BazarrConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: arrv1alpha1.BazarrConfigSpec{
					Sonarr: arrv1alpha1.BazarrConnectionSpec{
						URL: "http://sonarr.example.com:8989",
						APIKeySecretRef: &arrv1alpha1.SecretKeySelector{
							Name: sonarrSecretName,
							Key:  "apiKey",
						},
					},
					Radarr: arrv1alpha1.BazarrConnectionSpec{
						URL: "http://radarr.example.com:7878",
						APIKeySecretRef: &arrv1alpha1.SecretKeySelector{
							Name: radarrSecretName,
							Key:  "apiKey",
						},
					},
				},
			}

			// Create reconciler
			reconciler = &BazarrConfigReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Helper: NewReconcileHelper(k8sClient),
			}
		})

		AfterEach(func() {
			// Clean up BazarrConfig
			resource := &arrv1alpha1.BazarrConfig{}
			err := k8sClient.Get(ctx, typeNamespaceName, resource)
			if err == nil {
				// Remove finalizers for cleanup
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}

			// Clean up secrets
			_ = k8sClient.Delete(ctx, sonarrSecret)
			_ = k8sClient.Delete(ctx, radarrSecret)
		})

		It("should add finalizer on creation", func() {
			By("Creating the BazarrConfig resource")
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("Reconciling the resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the finalizer was added")
			updatedConfig := &arrv1alpha1.BazarrConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespaceName, updatedConfig)
				if err != nil {
					return false
				}
				for _, f := range updatedConfig.Finalizers {
					if f == bazarrFinalizer {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should set Ready=False when Sonarr secret is missing", func() {
			By("Creating BazarrConfig with non-existent Sonarr secret")
			bazarrConfig.Spec.Sonarr.APIKeySecretRef.Name = "non-existent-secret"
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to process")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(HaveOccurred())

			By("Checking that Ready condition is False")
			updatedConfig := &arrv1alpha1.BazarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(HasCondition(updatedConfig.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse)).To(BeTrue())
		})

		It("should set Ready=False when Radarr secret is missing", func() {
			By("Creating BazarrConfig with non-existent Radarr secret")
			bazarrConfig.Spec.Radarr.APIKeySecretRef.Name = "non-existent-secret"
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to process")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(HaveOccurred())

			By("Checking that Ready condition is False")
			updatedConfig := &arrv1alpha1.BazarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(HasCondition(updatedConfig.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse)).To(BeTrue())
		})

		It("should skip reconciliation when suspended", func() {
			By("Creating BazarrConfig with reconciliation suspended")
			bazarrConfig.Spec.Reconciliation = &arrv1alpha1.ReconciliationSpec{
				Suspend: true,
			}
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("Reconciling the resource")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that reconciliation was skipped (no requeue)")
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("Checking that no finalizer was added")
			updatedConfig := &arrv1alpha1.BazarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Finalizers).To(BeEmpty())
		})

		It("should generate ConfigMap when configMapRef is set", func() {
			By("Creating BazarrConfig with configMapRef")
			configMapName := "bazarr-config"
			bazarrConfig.Spec.ConfigMapRef = &arrv1alpha1.LocalObjectReference{
				Name: configMapName,
			}
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to generate config")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that ConfigMap was created")
			configMap := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      configMapName,
					Namespace: namespace,
				}, configMap)
			}, timeout, interval).Should(Succeed())

			By("Checking that ConfigMap has config.yaml key")
			Expect(configMap.Data).To(HaveKey("config.yaml"))

			By("Checking that status shows config generated")
			updatedConfig := &arrv1alpha1.BazarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.ConfigGenerated).To(BeTrue())
		})

		It("should track SonarrConnected and RadarrConnected status", func() {
			By("Creating BazarrConfig")
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to process")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking connection status")
			updatedConfig := &arrv1alpha1.BazarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			// These should be true because secrets were resolved successfully
			Expect(updatedConfig.Status.SonarrConnected).To(BeTrue())
			Expect(updatedConfig.Status.RadarrConnected).To(BeTrue())
		})

		It("should set LastReconcile timestamp", func() {
			By("Creating BazarrConfig")
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to process")
			beforeReconcile := time.Now()
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking LastReconcile was set")
			updatedConfig := &arrv1alpha1.BazarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.LastReconcile).NotTo(BeNil())
			Expect(updatedConfig.Status.LastReconcile.Time.After(beforeReconcile.Add(-1 * time.Second))).To(BeTrue())
		})

		It("should cleanup ConfigMap on deletion", func() {
			By("Creating BazarrConfig with configMapRef")
			configMapName := "bazarr-config-cleanup"
			bazarrConfig.Spec.ConfigMapRef = &arrv1alpha1.LocalObjectReference{
				Name: configMapName,
			}
			Expect(k8sClient.Create(ctx, bazarrConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to generate config")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ConfigMap exists")
			configMap := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      configMapName,
					Namespace: namespace,
				}, configMap)
			}, timeout, interval).Should(Succeed())

			By("Deleting the BazarrConfig")
			Expect(k8sClient.Delete(ctx, bazarrConfig)).To(Succeed())

			By("Reconciling the deletion")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ConfigMap was deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      configMapName,
					Namespace: namespace,
				}, configMap)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("should handle resource not found gracefully", func() {
			By("Reconciling a non-existent resource")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
})
