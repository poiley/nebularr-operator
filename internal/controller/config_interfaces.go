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
	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Finalizer names for each config type
const (
	SonarrFinalizer  = "sonarrconfig.arr.rinzler.cloud/finalizer"
	RadarrFinalizer  = "radarrconfig.arr.rinzler.cloud/finalizer"
	LidarrFinalizer  = "lidarrconfig.arr.rinzler.cloud/finalizer"
	ReadarrFinalizer = "readarrconfig.arr.rinzler.cloud/finalizer"
)

// -----------------------------------------------------------------------------
// SonarrConfig ArrConfigObject implementation
// -----------------------------------------------------------------------------

// SonarrConfigAdapter wraps SonarrConfig to implement ArrConfigObject
type SonarrConfigAdapter struct {
	*arrv1alpha1.SonarrConfig
}

func (a *SonarrConfigAdapter) GetConnectionSpec() *arrv1alpha1.ConnectionSpec {
	return &a.Spec.Connection
}

func (a *SonarrConfigAdapter) GetReconciliationSpec() *arrv1alpha1.ReconciliationSpec {
	return a.Spec.Reconciliation
}

func (a *SonarrConfigAdapter) GetDownloadClients() []arrv1alpha1.DownloadClientSpec {
	return a.Spec.DownloadClients
}

func (a *SonarrConfigAdapter) GetIndexersSpec() *arrv1alpha1.IndexersSpec {
	return a.Spec.Indexers
}

func (a *SonarrConfigAdapter) GetImportLists() []arrv1alpha1.ImportListSpec {
	return a.Spec.ImportLists
}

func (a *SonarrConfigAdapter) GetAuthenticationSpec() *arrv1alpha1.AuthenticationSpec {
	return a.Spec.Authentication
}

func (a *SonarrConfigAdapter) GetStatusWrapper() ConfigStatus {
	return &SonarrStatusWrapper{Status: &a.Status}
}

func (a *SonarrConfigAdapter) GetHealthStatusPtr() **arrv1alpha1.HealthStatus {
	return &a.Status.Health
}

func (a *SonarrConfigAdapter) GetAppType() string {
	return adapters.AppSonarr
}

func (a *SonarrConfigAdapter) GetFinalizerName() string {
	return SonarrFinalizer
}

func (a *SonarrConfigAdapter) ShouldRegisterWithProwlarr() bool {
	return true
}

func (a *SonarrConfigAdapter) GetObject() client.Object {
	return a.SonarrConfig
}

// SonarrConfigFetcher implements ConfigFetcher for SonarrConfig
type SonarrConfigFetcher struct{}

func (f SonarrConfigFetcher) NewEmpty() client.Object {
	return &arrv1alpha1.SonarrConfig{}
}

func (f SonarrConfigFetcher) Wrap(obj client.Object) ArrConfigObject {
	return &SonarrConfigAdapter{SonarrConfig: obj.(*arrv1alpha1.SonarrConfig)}
}

// -----------------------------------------------------------------------------
// RadarrConfig ArrConfigObject implementation
// -----------------------------------------------------------------------------

// RadarrConfigAdapter wraps RadarrConfig to implement ArrConfigObject
type RadarrConfigAdapter struct {
	*arrv1alpha1.RadarrConfig
}

func (a *RadarrConfigAdapter) GetConnectionSpec() *arrv1alpha1.ConnectionSpec {
	return &a.Spec.Connection
}

func (a *RadarrConfigAdapter) GetReconciliationSpec() *arrv1alpha1.ReconciliationSpec {
	return a.Spec.Reconciliation
}

func (a *RadarrConfigAdapter) GetDownloadClients() []arrv1alpha1.DownloadClientSpec {
	return a.Spec.DownloadClients
}

func (a *RadarrConfigAdapter) GetIndexersSpec() *arrv1alpha1.IndexersSpec {
	return a.Spec.Indexers
}

func (a *RadarrConfigAdapter) GetImportLists() []arrv1alpha1.ImportListSpec {
	return a.Spec.ImportLists
}

func (a *RadarrConfigAdapter) GetAuthenticationSpec() *arrv1alpha1.AuthenticationSpec {
	return a.Spec.Authentication
}

func (a *RadarrConfigAdapter) GetStatusWrapper() ConfigStatus {
	return &RadarrStatusWrapper{Status: &a.Status}
}

func (a *RadarrConfigAdapter) GetHealthStatusPtr() **arrv1alpha1.HealthStatus {
	return &a.Status.Health
}

func (a *RadarrConfigAdapter) GetAppType() string {
	return adapters.AppRadarr
}

func (a *RadarrConfigAdapter) GetFinalizerName() string {
	return RadarrFinalizer
}

func (a *RadarrConfigAdapter) ShouldRegisterWithProwlarr() bool {
	return true
}

func (a *RadarrConfigAdapter) GetObject() client.Object {
	return a.RadarrConfig
}

// RadarrConfigFetcher implements ConfigFetcher for RadarrConfig
type RadarrConfigFetcher struct{}

func (f RadarrConfigFetcher) NewEmpty() client.Object {
	return &arrv1alpha1.RadarrConfig{}
}

func (f RadarrConfigFetcher) Wrap(obj client.Object) ArrConfigObject {
	return &RadarrConfigAdapter{RadarrConfig: obj.(*arrv1alpha1.RadarrConfig)}
}

// -----------------------------------------------------------------------------
// LidarrConfig ArrConfigObject implementation
// -----------------------------------------------------------------------------

// LidarrConfigAdapter wraps LidarrConfig to implement ArrConfigObject
type LidarrConfigAdapter struct {
	*arrv1alpha1.LidarrConfig
}

func (a *LidarrConfigAdapter) GetConnectionSpec() *arrv1alpha1.ConnectionSpec {
	return &a.Spec.Connection
}

func (a *LidarrConfigAdapter) GetReconciliationSpec() *arrv1alpha1.ReconciliationSpec {
	return a.Spec.Reconciliation
}

func (a *LidarrConfigAdapter) GetDownloadClients() []arrv1alpha1.DownloadClientSpec {
	return a.Spec.DownloadClients
}

func (a *LidarrConfigAdapter) GetIndexersSpec() *arrv1alpha1.IndexersSpec {
	return a.Spec.Indexers
}

func (a *LidarrConfigAdapter) GetImportLists() []arrv1alpha1.ImportListSpec {
	return a.Spec.ImportLists
}

func (a *LidarrConfigAdapter) GetAuthenticationSpec() *arrv1alpha1.AuthenticationSpec {
	return a.Spec.Authentication
}

func (a *LidarrConfigAdapter) GetStatusWrapper() ConfigStatus {
	return &LidarrStatusWrapper{Status: &a.Status}
}

func (a *LidarrConfigAdapter) GetHealthStatusPtr() **arrv1alpha1.HealthStatus {
	return &a.Status.Health
}

func (a *LidarrConfigAdapter) GetAppType() string {
	return adapters.AppLidarr
}

func (a *LidarrConfigAdapter) GetFinalizerName() string {
	return LidarrFinalizer
}

func (a *LidarrConfigAdapter) ShouldRegisterWithProwlarr() bool {
	// Lidarr uses ProwlarrCoordinatorReconciler for registration to avoid duplicates
	return false
}

func (a *LidarrConfigAdapter) GetObject() client.Object {
	return a.LidarrConfig
}

// LidarrConfigFetcher implements ConfigFetcher for LidarrConfig
type LidarrConfigFetcher struct{}

func (f LidarrConfigFetcher) NewEmpty() client.Object {
	return &arrv1alpha1.LidarrConfig{}
}

func (f LidarrConfigFetcher) Wrap(obj client.Object) ArrConfigObject {
	return &LidarrConfigAdapter{LidarrConfig: obj.(*arrv1alpha1.LidarrConfig)}
}

// -----------------------------------------------------------------------------
// ReadarrConfig ArrConfigObject implementation
// -----------------------------------------------------------------------------

// ReadarrConfigAdapter wraps ReadarrConfig to implement ArrConfigObject
type ReadarrConfigAdapter struct {
	*arrv1alpha1.ReadarrConfig
}

func (a *ReadarrConfigAdapter) GetConnectionSpec() *arrv1alpha1.ConnectionSpec {
	return &a.Spec.Connection
}

func (a *ReadarrConfigAdapter) GetReconciliationSpec() *arrv1alpha1.ReconciliationSpec {
	return a.Spec.Reconciliation
}

func (a *ReadarrConfigAdapter) GetDownloadClients() []arrv1alpha1.DownloadClientSpec {
	return a.Spec.DownloadClients
}

func (a *ReadarrConfigAdapter) GetIndexersSpec() *arrv1alpha1.IndexersSpec {
	return a.Spec.Indexers
}

func (a *ReadarrConfigAdapter) GetImportLists() []arrv1alpha1.ImportListSpec {
	return a.Spec.ImportLists
}

func (a *ReadarrConfigAdapter) GetAuthenticationSpec() *arrv1alpha1.AuthenticationSpec {
	return a.Spec.Authentication
}

func (a *ReadarrConfigAdapter) GetStatusWrapper() ConfigStatus {
	return &ReadarrStatusWrapper{Status: &a.Status}
}

func (a *ReadarrConfigAdapter) GetHealthStatusPtr() **arrv1alpha1.HealthStatus {
	return &a.Status.Health
}

func (a *ReadarrConfigAdapter) GetAppType() string {
	return adapters.AppReadarr
}

func (a *ReadarrConfigAdapter) GetFinalizerName() string {
	return ReadarrFinalizer
}

func (a *ReadarrConfigAdapter) ShouldRegisterWithProwlarr() bool {
	return true
}

func (a *ReadarrConfigAdapter) GetObject() client.Object {
	return a.ReadarrConfig
}

// ReadarrConfigFetcher implements ConfigFetcher for ReadarrConfig
type ReadarrConfigFetcher struct{}

func (f ReadarrConfigFetcher) NewEmpty() client.Object {
	return &arrv1alpha1.ReadarrConfig{}
}

func (f ReadarrConfigFetcher) Wrap(obj client.Object) ArrConfigObject {
	return &ReadarrConfigAdapter{ReadarrConfig: obj.(*arrv1alpha1.ReadarrConfig)}
}
