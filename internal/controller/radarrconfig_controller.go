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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	_ "github.com/poiley/nebularr-operator/internal/adapters/radarr" // Register radarr adapter
	"github.com/poiley/nebularr-operator/internal/compiler"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// RadarrConfigReconciler reconciles a RadarrConfig object
type RadarrConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Compiler *compiler.Compiler
	Helper   *ReconcileHelper
	Recorder record.EventRecorder

	// generic holds the shared reconciliation logic
	generic *GenericArrReconciler
}

// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=radarrconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=radarrconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=radarrconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *RadarrConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ensureInitialized()
	return r.generic.FetchAndReconcile(ctx, req, RadarrConfigFetcher{})
}

// ensureInitialized initializes the generic reconciler if not already done.
func (r *RadarrConfigReconciler) ensureInitialized() {
	if r.generic != nil {
		return
	}
	if r.Compiler == nil {
		r.Compiler = compiler.New()
	}
	if r.Helper == nil {
		r.Helper = NewReconcileHelper(r.Client)
	}
	r.generic = &GenericArrReconciler{
		Client:   r.Client,
		Scheme:   r.Scheme,
		Compiler: r.Compiler,
		Helper:   r.Helper,
		Recorder: r.Recorder,
		CompileConfig: func(ctx context.Context, c *compiler.Compiler, config ArrConfigObject, secrets map[string]string, caps *adapters.Capabilities) (*irv1.IR, error) {
			radarrConfig := config.(*RadarrConfigAdapter).RadarrConfig
			return c.CompileRadarrConfig(ctx, radarrConfig, secrets, caps)
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RadarrConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ensureInitialized()

	return ctrl.NewControllerManagedBy(mgr).
		For(&arrv1alpha1.RadarrConfig{}).
		Named("radarrconfig").
		Complete(r)
}
