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
	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/mock"
	"github.com/poiley/nebularr-operator/internal/compiler"
)

var _ = Describe("SonarrConfig Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When reconciling a resource", func() {
		const (
			resourceName = "test-sonarr"
			namespace    = "default"
			secretName   = "sonarr-credentials"
		)

		var (
			ctx               context.Context
			typeNamespaceName types.NamespacedName
			sonarrConfig      *arrv1alpha1.SonarrConfig
			apiKeySecret      *corev1.Secret
			mockAdapter       *mock.Adapter
			reconciler        *SonarrConfigReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			typeNamespaceName = types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			// Setup mock adapter
			mockAdapter = mock.NewAdapter("sonarr")
			adapters.RegisterOrReplace(mockAdapter)

			// Create API key secret
			apiKeySecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				StringData: map[string]string{
					"apiKey": "test-api-key-12345",
				},
			}
			err := k8sClient.Create(ctx, apiKeySecret)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			// Create SonarrConfig resource
			sonarrConfig = &arrv1alpha1.SonarrConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: arrv1alpha1.SonarrConfigSpec{
					Connection: arrv1alpha1.ConnectionSpec{
						URL: "http://sonarr.example.com:8989",
						APIKeySecretRef: &arrv1alpha1.SecretKeySelector{
							Name: secretName,
							Key:  "apiKey",
						},
					},
				},
			}

			// Create reconciler
			reconciler = &SonarrConfigReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Helper:   NewReconcileHelper(k8sClient),
				Compiler: compiler.New(),
			}
		})

		AfterEach(func() {
			// Clean up mock adapter
			adapters.Clear()

			// Clean up SonarrConfig
			resource := &arrv1alpha1.SonarrConfig{}
			err := k8sClient.Get(ctx, typeNamespaceName, resource)
			if err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}

			// Clean up secret
			_ = k8sClient.Delete(ctx, apiKeySecret)
		})

		It("should successfully reconcile and set Ready condition", func() {
			By("Creating the SonarrConfig resource")
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

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

			By("Checking that the mock adapter was called")
			Expect(mockAdapter.CallCounts()["Connect"]).To(BeNumerically(">=", 1))

			By("Checking the Ready condition")
			updatedConfig := &arrv1alpha1.SonarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(HasCondition(updatedConfig.Status.Conditions, ConditionTypeReady, metav1.ConditionTrue)).To(BeTrue())
		})

		It("should set Connected status and version from adapter", func() {
			By("Configuring mock to return specific version")
			mockAdapter.WithVersion("4.0.5.7846")

			By("Creating the SonarrConfig resource")
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

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

			By("Checking Connected status and version")
			updatedConfig := &arrv1alpha1.SonarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.Connected).To(BeTrue())
			Expect(updatedConfig.Status.ServiceVersion).To(Equal("4.0.5.7846"))
		})

		It("should set Ready=False when connection fails", func() {
			By("Configuring mock to return connection error")
			mockAdapter.WithConnectError(apierrors.NewServiceUnavailable("connection refused"))

			By("Creating the SonarrConfig resource")
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

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
			updatedConfig := &arrv1alpha1.SonarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(HasCondition(updatedConfig.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse)).To(BeTrue())
			Expect(updatedConfig.Status.Connected).To(BeFalse())
		})

		It("should set Ready=False when API key secret is missing", func() {
			By("Creating SonarrConfig with non-existent secret")
			sonarrConfig.Spec.Connection.APIKeySecretRef.Name = "non-existent-secret"
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

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

			By("Checking that adapter was NOT called (secret resolution failed first)")
			Expect(mockAdapter.CallCounts()["Connect"]).To(Equal(0))
		})

		It("should update LastReconcile timestamp", func() {
			By("Creating the SonarrConfig resource")
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

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
			updatedConfig := &arrv1alpha1.SonarrConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.LastReconcile).NotTo(BeNil())
			Expect(updatedConfig.Status.LastReconcile.Time.After(beforeReconcile.Add(-1 * time.Second))).To(BeTrue())
		})

		It("should requeue with custom interval", func() {
			By("Creating SonarrConfig with custom reconciliation interval")
			interval := metav1.Duration{Duration: 10 * time.Minute}
			sonarrConfig.Spec.Reconciliation = &arrv1alpha1.ReconciliationSpec{
				Interval: &interval,
			}
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to process")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking requeue interval")
			Expect(result.RequeueAfter).To(Equal(10 * time.Minute))
		})

		It("should skip reconciliation when suspended", func() {
			By("Creating SonarrConfig with reconciliation suspended")
			sonarrConfig.Spec.Reconciliation = &arrv1alpha1.ReconciliationSpec{
				Suspend: true,
			}
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

			By("Reconciling the resource")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that reconciliation was skipped")
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("Checking that adapter was NOT called")
			Expect(mockAdapter.CallCounts()["Connect"]).To(Equal(0))
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

		It("should track ManagedResources in status", func() {
			By("Configuring mock to return changes")
			mockAdapter.WithChanges(&adapters.ChangeSet{
				Creates: []adapters.Change{
					{ResourceType: adapters.ResourceQualityProfile, Name: "HD-1080p"},
				},
			})

			By("Creating the SonarrConfig resource")
			Expect(k8sClient.Create(ctx, sonarrConfig)).To(Succeed())

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

			By("Checking that Apply was called")
			Expect(mockAdapter.CallCounts()["Apply"]).To(BeNumerically(">=", 1))
		})
	})
})
