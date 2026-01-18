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

package downloadstack

import (
	"context"
	"sync"
)

// MockTransmissionClient is a test double for TransmissionClient.
// All methods are configurable via function fields.
type MockTransmissionClient struct {
	// Configurable function implementations
	TestConnectionFunc  func(ctx context.Context) error
	GetSessionFunc      func(ctx context.Context) (*TransmissionSession, error)
	SetSessionFunc      func(ctx context.Context, settings map[string]interface{}) error
	GetSessionStatsFunc func(ctx context.Context) (map[string]interface{}, error)
	UpdateBlocklistFunc func(ctx context.Context) error

	// Call tracking
	mu                   sync.Mutex
	TestConnectionCalls  int
	GetSessionCalls      int
	SetSessionCalls      []map[string]interface{}
	GetSessionStatsCalls int
	UpdateBlocklistCalls int
}

// Ensure MockTransmissionClient implements the interface
var _ TransmissionClientInterface = (*MockTransmissionClient)(nil)

// NewMockTransmissionClient creates a new mock with default happy-path implementations.
func NewMockTransmissionClient() *MockTransmissionClient {
	return &MockTransmissionClient{
		SetSessionCalls: make([]map[string]interface{}, 0),
	}
}

// TestConnection tests the connection to Transmission.
func (m *MockTransmissionClient) TestConnection(ctx context.Context) error {
	m.mu.Lock()
	m.TestConnectionCalls++
	m.mu.Unlock()

	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx)
	}
	return nil
}

// GetSession gets session/settings information.
func (m *MockTransmissionClient) GetSession(ctx context.Context) (*TransmissionSession, error) {
	m.mu.Lock()
	m.GetSessionCalls++
	m.mu.Unlock()

	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(ctx)
	}

	// Default: return a valid session
	return &TransmissionSession{
		Version:             "4.0.0",
		DownloadDir:         "/downloads",
		SpeedLimitDown:      0,
		SpeedLimitUp:        0,
		PeerLimitGlobal:     200,
		PeerLimitPerTorrent: 50,
		PeerPort:            51413,
		Encryption:          "preferred",
		DhtEnabled:          true,
		PexEnabled:          true,
		UtpEnabled:          true,
	}, nil
}

// SetSession updates session settings.
func (m *MockTransmissionClient) SetSession(ctx context.Context, settings map[string]interface{}) error {
	m.mu.Lock()
	m.SetSessionCalls = append(m.SetSessionCalls, settings)
	m.mu.Unlock()

	if m.SetSessionFunc != nil {
		return m.SetSessionFunc(ctx, settings)
	}
	return nil
}

// GetSessionStats gets session statistics.
func (m *MockTransmissionClient) GetSessionStats(ctx context.Context) (map[string]interface{}, error) {
	m.mu.Lock()
	m.GetSessionStatsCalls++
	m.mu.Unlock()

	if m.GetSessionStatsFunc != nil {
		return m.GetSessionStatsFunc(ctx)
	}

	// Default: return empty stats
	return map[string]interface{}{
		"activeTorrentCount": 0,
		"downloadSpeed":      0,
		"uploadSpeed":        0,
	}, nil
}

// UpdateBlocklist updates the blocklist.
func (m *MockTransmissionClient) UpdateBlocklist(ctx context.Context) error {
	m.mu.Lock()
	m.UpdateBlocklistCalls++
	m.mu.Unlock()

	if m.UpdateBlocklistFunc != nil {
		return m.UpdateBlocklistFunc(ctx)
	}
	return nil
}

// Reset clears all call tracking data.
func (m *MockTransmissionClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TestConnectionCalls = 0
	m.GetSessionCalls = 0
	m.SetSessionCalls = make([]map[string]interface{}, 0)
	m.GetSessionStatsCalls = 0
	m.UpdateBlocklistCalls = 0
}

// WithConnectionError configures the mock to return an error on TestConnection.
func (m *MockTransmissionClient) WithConnectionError(err error) *MockTransmissionClient {
	m.TestConnectionFunc = func(ctx context.Context) error {
		return err
	}
	return m
}

// WithSession configures the mock to return a specific session.
func (m *MockTransmissionClient) WithSession(session *TransmissionSession) *MockTransmissionClient {
	m.GetSessionFunc = func(ctx context.Context) (*TransmissionSession, error) {
		return session, nil
	}
	return m
}

// WithSetSessionError configures the mock to return an error on SetSession.
func (m *MockTransmissionClient) WithSetSessionError(err error) *MockTransmissionClient {
	m.SetSessionFunc = func(ctx context.Context, settings map[string]interface{}) error {
		return err
	}
	return m
}
