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

// Package mock provides mock implementations of adapters for testing.
package mock

import (
	"context"
	"sync"
	"time"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Adapter implements adapters.Adapter for testing purposes.
// All methods are configurable via function fields.
// If a function field is nil, a sensible default is used.
type Adapter struct {
	// AppName is the app type this adapter handles (e.g., "radarr", "sonarr")
	AppName string

	// AdapterName is the unique name for this adapter instance
	AdapterName string

	// Configurable function implementations
	ConnectFunc      func(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error)
	DiscoverFunc     func(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error)
	CurrentStateFunc func(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error)
	DiffFunc         func(current, desired *irv1.IR, caps *adapters.Capabilities) (*adapters.ChangeSet, error)
	ApplyFunc        func(ctx context.Context, conn *irv1.ConnectionIR, changes *adapters.ChangeSet) (*adapters.ApplyResult, error)

	// Optional interface implementations
	ApplyDirectFunc func(ctx context.Context, conn *irv1.ConnectionIR, ir *irv1.IR) (*adapters.ApplyResult, error)
	GetHealthFunc   func(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.HealthStatus, error)

	// Call tracking for assertions
	mu                sync.Mutex
	ConnectCalls      []ConnectCall
	DiscoverCalls     []DiscoverCall
	CurrentStateCalls []CurrentStateCall
	DiffCalls         []DiffCall
	ApplyCalls        []ApplyCall
	ApplyDirectCalls  []ApplyDirectCall
	GetHealthCalls    []GetHealthCall
}

// Call tracking types
type ConnectCall struct {
	Conn *irv1.ConnectionIR
}

type DiscoverCall struct {
	Conn *irv1.ConnectionIR
}

type CurrentStateCall struct {
	Conn *irv1.ConnectionIR
}

type DiffCall struct {
	Current *irv1.IR
	Desired *irv1.IR
	Caps    *adapters.Capabilities
}

type ApplyCall struct {
	Conn    *irv1.ConnectionIR
	Changes *adapters.ChangeSet
}

type ApplyDirectCall struct {
	Conn *irv1.ConnectionIR
	IR   *irv1.IR
}

type GetHealthCall struct {
	Conn *irv1.ConnectionIR
}

// Ensure Adapter implements the required interfaces
var (
	_ adapters.Adapter       = (*Adapter)(nil)
	_ adapters.DirectApplier = (*Adapter)(nil)
	_ adapters.HealthChecker = (*Adapter)(nil)
)

// NewAdapter creates a new mock adapter with default happy-path implementations.
func NewAdapter(appName string) *Adapter {
	return &Adapter{
		AppName:     appName,
		AdapterName: "mock-" + appName,
	}
}

// Name returns the adapter's unique identifier.
func (m *Adapter) Name() string {
	if m.AdapterName != "" {
		return m.AdapterName
	}
	return "mock-" + m.AppName
}

// SupportedApp returns the app type this adapter handles.
func (m *Adapter) SupportedApp() string {
	return m.AppName
}

// Connect tests connectivity and retrieves service info.
func (m *Adapter) Connect(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
	m.mu.Lock()
	m.ConnectCalls = append(m.ConnectCalls, ConnectCall{Conn: conn})
	m.mu.Unlock()

	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx, conn)
	}

	// Default: successful connection
	return &adapters.ServiceInfo{
		Version:   "1.0.0-mock",
		StartTime: time.Now(),
	}, nil
}

// Discover queries the service for its capabilities.
func (m *Adapter) Discover(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error) {
	m.mu.Lock()
	m.DiscoverCalls = append(m.DiscoverCalls, DiscoverCall{Conn: conn})
	m.mu.Unlock()

	if m.DiscoverFunc != nil {
		return m.DiscoverFunc(ctx, conn)
	}

	// Default: basic capabilities
	return &adapters.Capabilities{
		DiscoveredAt:        time.Now(),
		Resolutions:         []string{"2160p", "1080p", "720p", "480p"},
		Sources:             []string{"bluray", "webdl", "webrip", "hdtv"},
		DownloadClientTypes: []string{"qbittorrent", "transmission", "sabnzbd"},
		IndexerTypes:        []string{"torznab", "newznab"},
	}, nil
}

// CurrentState retrieves the current managed state from the service.
func (m *Adapter) CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error) {
	m.mu.Lock()
	m.CurrentStateCalls = append(m.CurrentStateCalls, CurrentStateCall{Conn: conn})
	m.mu.Unlock()

	if m.CurrentStateFunc != nil {
		return m.CurrentStateFunc(ctx, conn)
	}

	// Default: empty state (nothing managed yet)
	return &irv1.IR{
		App: m.AppName,
	}, nil
}

// Diff computes the changes needed to move from current to desired state.
func (m *Adapter) Diff(current, desired *irv1.IR, caps *adapters.Capabilities) (*adapters.ChangeSet, error) {
	m.mu.Lock()
	m.DiffCalls = append(m.DiffCalls, DiffCall{Current: current, Desired: desired, Caps: caps})
	m.mu.Unlock()

	if m.DiffFunc != nil {
		return m.DiffFunc(current, desired, caps)
	}

	// Default: no changes needed
	return &adapters.ChangeSet{}, nil
}

// Apply executes the changes against the service.
func (m *Adapter) Apply(ctx context.Context, conn *irv1.ConnectionIR, changes *adapters.ChangeSet) (*adapters.ApplyResult, error) {
	m.mu.Lock()
	m.ApplyCalls = append(m.ApplyCalls, ApplyCall{Conn: conn, Changes: changes})
	m.mu.Unlock()

	if m.ApplyFunc != nil {
		return m.ApplyFunc(ctx, conn, changes)
	}

	// Default: all changes applied successfully
	totalChanges := len(changes.Creates) + len(changes.Updates) + len(changes.Deletes)
	return &adapters.ApplyResult{
		Applied: totalChanges,
		Failed:  0,
		Skipped: 0,
	}, nil
}

// ApplyDirect applies configuration directly from IR (DirectApplier interface).
func (m *Adapter) ApplyDirect(ctx context.Context, conn *irv1.ConnectionIR, ir *irv1.IR) (*adapters.ApplyResult, error) {
	m.mu.Lock()
	m.ApplyDirectCalls = append(m.ApplyDirectCalls, ApplyDirectCall{Conn: conn, IR: ir})
	m.mu.Unlock()

	if m.ApplyDirectFunc != nil {
		return m.ApplyDirectFunc(ctx, conn, ir)
	}

	// Default: success with count based on IR contents
	applied := 0
	if len(ir.ImportLists) > 0 {
		applied += len(ir.ImportLists)
	}
	if ir.MediaManagement != nil {
		applied++
	}
	if ir.Authentication != nil {
		applied++
	}

	return &adapters.ApplyResult{
		Applied: applied,
		Failed:  0,
		Skipped: 0,
	}, nil
}

// GetHealth fetches the current health status (HealthChecker interface).
func (m *Adapter) GetHealth(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.HealthStatus, error) {
	m.mu.Lock()
	m.GetHealthCalls = append(m.GetHealthCalls, GetHealthCall{Conn: conn})
	m.mu.Unlock()

	if m.GetHealthFunc != nil {
		return m.GetHealthFunc(ctx, conn)
	}

	// Default: healthy with no issues
	return &irv1.HealthStatus{
		Healthy: true,
		Issues:  []irv1.HealthIssue{},
	}, nil
}

// Reset clears all call tracking data.
func (m *Adapter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ConnectCalls = nil
	m.DiscoverCalls = nil
	m.CurrentStateCalls = nil
	m.DiffCalls = nil
	m.ApplyCalls = nil
	m.ApplyDirectCalls = nil
	m.GetHealthCalls = nil
}

// CallCounts returns the number of times each method was called.
func (m *Adapter) CallCounts() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return map[string]int{
		"Connect":      len(m.ConnectCalls),
		"Discover":     len(m.DiscoverCalls),
		"CurrentState": len(m.CurrentStateCalls),
		"Diff":         len(m.DiffCalls),
		"Apply":        len(m.ApplyCalls),
		"ApplyDirect":  len(m.ApplyDirectCalls),
		"GetHealth":    len(m.GetHealthCalls),
	}
}

// WithConnectError returns the adapter configured to return an error on Connect.
func (m *Adapter) WithConnectError(err error) *Adapter {
	m.ConnectFunc = func(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
		return nil, err
	}
	return m
}

// WithVersion returns the adapter configured to return a specific version.
func (m *Adapter) WithVersion(version string) *Adapter {
	m.ConnectFunc = func(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
		return &adapters.ServiceInfo{
			Version:   version,
			StartTime: time.Now(),
		}, nil
	}
	return m
}

// WithCurrentState returns the adapter configured to return a specific current state.
func (m *Adapter) WithCurrentState(ir *irv1.IR) *Adapter {
	m.CurrentStateFunc = func(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error) {
		return ir, nil
	}
	return m
}

// WithChanges returns the adapter configured to return specific changes from Diff.
func (m *Adapter) WithChanges(changes *adapters.ChangeSet) *Adapter {
	m.DiffFunc = func(current, desired *irv1.IR, caps *adapters.Capabilities) (*adapters.ChangeSet, error) {
		return changes, nil
	}
	return m
}

// WithApplyError returns the adapter configured to return an error on Apply.
func (m *Adapter) WithApplyError(err error) *Adapter {
	m.ApplyFunc = func(ctx context.Context, conn *irv1.ConnectionIR, changes *adapters.ChangeSet) (*adapters.ApplyResult, error) {
		return &adapters.ApplyResult{
			Applied: 0,
			Failed:  changes.TotalChanges(),
		}, err
	}
	return m
}

// WithHealthIssues returns the adapter configured to return specific health issues.
func (m *Adapter) WithHealthIssues(issues []irv1.HealthIssue) *Adapter {
	m.GetHealthFunc = func(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.HealthStatus, error) {
		healthy := true
		for _, issue := range issues {
			if issue.Type == irv1.HealthIssueTypeError {
				healthy = false
				break
			}
		}
		return &irv1.HealthStatus{
			Healthy: healthy,
			Issues:  issues,
		}, nil
	}
	return m
}

// WithDiscoverError returns the adapter configured to return an error on Discover.
func (m *Adapter) WithDiscoverError(err error) *Adapter {
	m.DiscoverFunc = func(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error) {
		return nil, err
	}
	return m
}

// WithDiscoverVersion configures the mock to also set version on Discover calls.
// This is useful since controllers typically use Discover to get service info.
func (m *Adapter) WithDiscoverVersion(version string) *Adapter {
	m.DiscoverFunc = func(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error) {
		return &adapters.Capabilities{
			DiscoveredAt:        time.Now(),
			Resolutions:         []string{"2160p", "1080p", "720p", "480p"},
			Sources:             []string{"bluray", "webdl", "webrip", "hdtv"},
			DownloadClientTypes: []string{"qbittorrent", "transmission", "sabnzbd"},
			IndexerTypes:        []string{"torznab", "newznab"},
		}, nil
	}
	// Also set version on Connect since ReconcileConfig calls it
	m.ConnectFunc = func(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
		return &adapters.ServiceInfo{
			Version:   version,
			StartTime: time.Now(),
		}, nil
	}
	return m
}
