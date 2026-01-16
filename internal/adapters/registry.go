package adapters

import (
	"fmt"
	"sync"
)

// registry holds all registered adapters
var (
	registry = make(map[string]Adapter)
	mu       sync.RWMutex
)

// Register adds an adapter to the registry.
// It panics if an adapter for the same app is already registered.
func Register(a Adapter) {
	mu.Lock()
	defer mu.Unlock()

	app := a.SupportedApp()
	if _, exists := registry[app]; exists {
		panic(fmt.Sprintf("adapter for app %q already registered", app))
	}
	registry[app] = a
}

// Get retrieves an adapter by app name.
// Returns nil, false if no adapter is registered for the app.
func Get(app string) (Adapter, bool) {
	mu.RLock()
	defer mu.RUnlock()

	a, ok := registry[app]
	return a, ok
}

// MustGet retrieves an adapter by app name.
// It panics if no adapter is registered for the app.
func MustGet(app string) Adapter {
	a, ok := Get(app)
	if !ok {
		panic(fmt.Sprintf("no adapter registered for app %q", app))
	}
	return a
}

// List returns all registered app names.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	apps := make([]string, 0, len(registry))
	for app := range registry {
		apps = append(apps, app)
	}
	return apps
}

// Count returns the number of registered adapters.
func Count() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(registry)
}

// Clear removes all registered adapters.
// This is primarily useful for testing.
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Adapter)
}
