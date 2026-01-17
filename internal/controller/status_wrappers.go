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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
)

// RadarrStatusWrapper wraps RadarrConfigStatus to implement ConfigStatus
type RadarrStatusWrapper struct {
	Status *arrv1alpha1.RadarrConfigStatus
}

func (w *RadarrStatusWrapper) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *RadarrStatusWrapper) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func (w *RadarrStatusWrapper) SetConnected(connected bool) {
	w.Status.Connected = connected
}

func (w *RadarrStatusWrapper) SetServiceVersion(version string) {
	w.Status.ServiceVersion = version
}

func (w *RadarrStatusWrapper) SetLastReconcile(t *metav1.Time) {
	w.Status.LastReconcile = t
}

func (w *RadarrStatusWrapper) SetLastAppliedHash(hash string) {
	w.Status.LastAppliedHash = hash
}

// SonarrStatusWrapper wraps SonarrConfigStatus to implement ConfigStatus
type SonarrStatusWrapper struct {
	Status *arrv1alpha1.SonarrConfigStatus
}

func (w *SonarrStatusWrapper) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *SonarrStatusWrapper) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func (w *SonarrStatusWrapper) SetConnected(connected bool) {
	w.Status.Connected = connected
}

func (w *SonarrStatusWrapper) SetServiceVersion(version string) {
	w.Status.ServiceVersion = version
}

func (w *SonarrStatusWrapper) SetLastReconcile(t *metav1.Time) {
	w.Status.LastReconcile = t
}

func (w *SonarrStatusWrapper) SetLastAppliedHash(hash string) {
	w.Status.LastAppliedHash = hash
}

// LidarrStatusWrapper wraps LidarrConfigStatus to implement ConfigStatus
type LidarrStatusWrapper struct {
	Status *arrv1alpha1.LidarrConfigStatus
}

func (w *LidarrStatusWrapper) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *LidarrStatusWrapper) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func (w *LidarrStatusWrapper) SetConnected(connected bool) {
	w.Status.Connected = connected
}

func (w *LidarrStatusWrapper) SetServiceVersion(version string) {
	w.Status.ServiceVersion = version
}

func (w *LidarrStatusWrapper) SetLastReconcile(t *metav1.Time) {
	w.Status.LastReconcile = t
}

func (w *LidarrStatusWrapper) SetLastAppliedHash(hash string) {
	w.Status.LastAppliedHash = hash
}

// ProwlarrStatusWrapper wraps ProwlarrConfigStatus to implement ConfigStatus
type ProwlarrStatusWrapper struct {
	Status *arrv1alpha1.ProwlarrConfigStatus
}

func (w *ProwlarrStatusWrapper) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *ProwlarrStatusWrapper) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func (w *ProwlarrStatusWrapper) SetConnected(connected bool) {
	w.Status.Connected = connected
}

func (w *ProwlarrStatusWrapper) SetServiceVersion(version string) {
	w.Status.ServiceVersion = version
}

func (w *ProwlarrStatusWrapper) SetLastReconcile(t *metav1.Time) {
	w.Status.LastReconcile = t
}

func (w *ProwlarrStatusWrapper) SetLastAppliedHash(hash string) {
	w.Status.LastAppliedHash = hash
}

// BazarrStatusWrapper wraps BazarrConfigStatus to implement ConfigStatus
// Note: Bazarr has a different status structure (no Connected/ServiceVersion)
type BazarrStatusWrapper struct {
	Status *arrv1alpha1.BazarrConfigStatus
}

func (w *BazarrStatusWrapper) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *BazarrStatusWrapper) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func (w *BazarrStatusWrapper) SetConnected(connected bool) {
	// Bazarr doesn't have a single Connected field
	// This is a no-op since Bazarr uses SonarrConnected/RadarrConnected instead
}

func (w *BazarrStatusWrapper) SetServiceVersion(version string) {
	// Bazarr doesn't track service version in the same way
}

func (w *BazarrStatusWrapper) SetLastReconcile(t *metav1.Time) {
	w.Status.LastReconcile = t
}

func (w *BazarrStatusWrapper) SetLastAppliedHash(hash string) {
	w.Status.LastAppliedHash = hash
}
