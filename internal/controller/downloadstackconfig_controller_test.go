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
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters/downloadstack"
)

var _ = Describe("DownloadStackConfig Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When reconciling a DownloadStackConfig resource", func() {
		const (
			resourceName      = "test-downloadstack"
			namespace         = "default"
			deploymentName    = "download-stack"
			gluetunSecretName = "vpn-credentials"
		)

		var (
			ctx               context.Context
			typeNamespaceName types.NamespacedName
			dsConfig          *arrv1alpha1.DownloadStackConfig
			gluetunSecret     *corev1.Secret
			deployment        *appsv1.Deployment
			mockTransmission  *downloadstack.MockTransmissionClient
			reconciler        *DownloadStackConfigReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			typeNamespaceName = types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			// Create mock Transmission client
			mockTransmission = downloadstack.NewMockTransmissionClient()

			// Create Gluetun credentials secret
			gluetunSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gluetunSecretName,
					Namespace: namespace,
				},
				StringData: map[string]string{
					"username": "vpn-user",
					"password": "vpn-pass",
				},
			}
			err := k8sClient.Create(ctx, gluetunSecret)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			// Create dummy Deployment that we'll reference
			deployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "download-stack"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "download-stack"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "transmission",
									Image: "transmission:latest",
								},
							},
						},
					},
				},
			}
			err = k8sClient.Create(ctx, deployment)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			// Create DownloadStackConfig resource
			dsConfig = &arrv1alpha1.DownloadStackConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: arrv1alpha1.DownloadStackConfigSpec{
					DeploymentRef: arrv1alpha1.LocalObjectReference{
						Name: deploymentName,
					},
					Gluetun: arrv1alpha1.GluetunSpec{
						Provider: arrv1alpha1.GluetunProviderSpec{
							Name: "mullvad",
							CredentialsSecretRef: &arrv1alpha1.CredentialsSecretRef{
								Name: gluetunSecretName,
							},
						},
						VPNType: "openvpn",
					},
					Transmission: &arrv1alpha1.TransmissionSpec{
						Connection: arrv1alpha1.TransmissionConnectionSpec{
							URL: "http://localhost:9091",
						},
					},
					RestartOnGluetunChange: true,
				},
			}

			// Create reconciler with mock Transmission client factory
			reconciler = &DownloadStackConfigReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Helper: NewReconcileHelper(k8sClient),
				TransmissionClientFactory: func(url, username, password string) downloadstack.TransmissionClientInterface {
					return mockTransmission
				},
			}
		})

		AfterEach(func() {
			// Clean up DownloadStackConfig
			resource := &arrv1alpha1.DownloadStackConfig{}
			err := k8sClient.Get(ctx, typeNamespaceName, resource)
			if err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}

			// Clean up secrets
			_ = k8sClient.Delete(ctx, gluetunSecret)

			// Clean up deployment
			_ = k8sClient.Delete(ctx, deployment)

			// Clean up any generated secrets
			generatedSecret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{
				Name:      resourceName + "-gluetun-env",
				Namespace: namespace,
			}, generatedSecret)
			_ = k8sClient.Delete(ctx, generatedSecret)
		})

		It("should add finalizer on creation", func() {
			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

			By("Reconciling the resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the finalizer was added")
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespaceName, updatedConfig)
				if err != nil {
					return false
				}
				for _, f := range updatedConfig.Finalizers {
					if f == downloadStackFinalizer {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should create Gluetun env Secret", func() {
			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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

			By("Checking that Gluetun Secret was created")
			gluetunEnvSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName + "-gluetun-env",
					Namespace: namespace,
				}, gluetunEnvSecret)
			}, timeout, interval).Should(Succeed())

			By("Checking Secret contains expected env vars")
			Expect(gluetunEnvSecret.Data).To(HaveKey("VPN_SERVICE_PROVIDER"))
			Expect(string(gluetunEnvSecret.Data["VPN_SERVICE_PROVIDER"])).To(Equal("mullvad"))
		})

		It("should set Ready=False when Gluetun credentials secret is missing", func() {
			By("Creating DownloadStackConfig with non-existent credentials")
			dsConfig.Spec.Gluetun.Provider.CredentialsSecretRef.Name = "non-existent-secret"
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(HasCondition(updatedConfig.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse)).To(BeTrue())
		})

		It("should set Ready=False when Transmission is unreachable", func() {
			By("Configuring mock to return connection error")
			mockTransmission.WithConnectionError(errors.New("connection refused"))

			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(HasCondition(updatedConfig.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse)).To(BeTrue())

			By("Checking that TransmissionConnected is false")
			Expect(updatedConfig.Status.TransmissionConnected).To(BeFalse())
		})

		It("should set TransmissionConnected=true and track version", func() {
			By("Configuring mock to return version")
			mockTransmission.WithSession(&downloadstack.TransmissionSession{
				Version: "4.0.5",
			})

			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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

			By("Checking TransmissionConnected and version")
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.TransmissionConnected).To(BeTrue())
			Expect(updatedConfig.Status.TransmissionVersion).To(Equal("4.0.5"))
		})

		It("should track GluetunConfigHash in status", func() {
			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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

			By("Checking that GluetunConfigHash is set")
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.GluetunConfigHash).NotTo(BeEmpty())
		})

		It("should skip reconciliation when suspended", func() {
			By("Creating DownloadStackConfig with reconciliation suspended")
			dsConfig.Spec.Reconciliation = &arrv1alpha1.ReconciliationSpec{
				Suspend: true,
			}
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

			By("Reconciling the resource")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that reconciliation was skipped")
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("Checking that no finalizer was added")
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Finalizers).To(BeEmpty())
		})

		It("should trigger Deployment restart when Gluetun config changes", func() {
			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

			By("First reconcile to add finalizer")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile to process initial config")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Getting initial config hash")
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			initialHash := updatedConfig.Status.GluetunConfigHash

			By("Updating the Gluetun config")
			updatedConfig.Spec.Gluetun.Server = &arrv1alpha1.GluetunServerSpec{
				Countries: []string{"Germany"},
			}
			Expect(k8sClient.Update(ctx, updatedConfig)).To(Succeed())

			By("Reconciling the updated resource")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that config hash changed")
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.GluetunConfigHash).NotTo(Equal(initialHash))

			By("Checking that Deployment has restart annotation")
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      deploymentName,
				Namespace: namespace,
			}, dep)).To(Succeed())
			Expect(dep.Spec.Template.Annotations).To(HaveKey(restartAnnotationKey))
		})

		It("should set GluetunSecretGenerated in status", func() {
			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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

			By("Checking GluetunSecretGenerated status")
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.GluetunSecretGenerated).To(BeTrue())
		})

		It("should set ObservedGeneration on successful reconcile", func() {
			By("Creating the DownloadStackConfig resource")
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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

			By("Checking ObservedGeneration matches Generation")
			updatedConfig := &arrv1alpha1.DownloadStackConfig{}
			Expect(k8sClient.Get(ctx, typeNamespaceName, updatedConfig)).To(Succeed())
			Expect(updatedConfig.Status.ObservedGeneration).To(Equal(updatedConfig.Generation))
		})

		It("should sync Transmission settings via mock client", func() {
			By("Creating DownloadStackConfig with speed limits")
			dsConfig.Spec.Transmission.Speed = &arrv1alpha1.TransmissionSpeedSpec{
				DownloadLimit:        10000,
				DownloadLimitEnabled: true,
			}
			Expect(k8sClient.Create(ctx, dsConfig)).To(Succeed())

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

			By("Checking that SetSession was called")
			Expect(mockTransmission.SetSessionCalls).To(HaveLen(1))

			By("Checking that speed limits were set")
			settings := mockTransmission.SetSessionCalls[0]
			Expect(settings).To(HaveKey("speed-limit-down"))
			Expect(settings["speed-limit-down"]).To(Equal(10000))
			Expect(settings["speed-limit-down-enabled"]).To(BeTrue())
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
