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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/mock"
)

// TestHelper provides utilities for controller tests.
type TestHelper struct {
	Client client.Client
}

// NewTestHelper creates a new TestHelper.
func NewTestHelper(c client.Client) *TestHelper {
	return &TestHelper{Client: c}
}

// SetupMockAdapter creates and registers a mock adapter for testing.
// Returns the mock adapter so tests can customize its behavior.
func SetupMockAdapter(appName string) *mock.Adapter {
	m := mock.NewAdapter(appName)
	adapters.RegisterOrReplace(m)
	return m
}

// CleanupAdapters removes all registered adapters.
// Should be called in AfterEach to ensure test isolation.
func CleanupAdapters() {
	adapters.Clear()
}

// CreateTestSecret creates a Secret for testing.
func (h *TestHelper) CreateTestSecret(ctx context.Context, namespace, name string, data map[string]string) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: data,
	}

	if err := h.Client.Create(ctx, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// CreateAPIKeySecret creates a Secret with an API key for testing.
func (h *TestHelper) CreateAPIKeySecret(ctx context.Context, namespace, name, apiKey string) (*corev1.Secret, error) {
	return h.CreateTestSecret(ctx, namespace, name, map[string]string{
		"apiKey": apiKey,
	})
}

// CreateCredentialsSecret creates a Secret with username/password for testing.
func (h *TestHelper) CreateCredentialsSecret(ctx context.Context, namespace, name, username, password string) (*corev1.Secret, error) {
	return h.CreateTestSecret(ctx, namespace, name, map[string]string{
		"username": username,
		"password": password,
	})
}

// DeleteTestResource deletes a resource if it exists.
func (h *TestHelper) DeleteTestResource(ctx context.Context, obj client.Object) error {
	if err := h.Client.Delete(ctx, obj); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// GetCondition returns the condition with the given type from a list of conditions.
func GetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// HasCondition checks if a condition with the given type and status exists.
func HasCondition(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus) bool {
	cond := GetCondition(conditions, conditionType)
	if cond == nil {
		return false
	}
	return cond.Status == status
}

// HasConditionWithReason checks if a condition with the given type, status, and reason exists.
func HasConditionWithReason(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus, reason string) bool {
	cond := GetCondition(conditions, conditionType)
	if cond == nil {
		return false
	}
	return cond.Status == status && cond.Reason == reason
}
