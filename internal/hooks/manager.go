package hooks

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
)

// Manager handles hook registration and execution
type Manager struct {
	mu     sync.RWMutex
	hooks  map[EventType][]hookEntry
	logger Logger
}

// hookEntry wraps a hook with its priority
type hookEntry struct {
	hook     Hook
	priority Priority
}

// Logger interface for hook logging
type Logger interface {
	Printf(format string, v ...interface{})
}

// defaultLogger uses the standard log package
type defaultLogger struct{}

func (l *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// NewManager creates a new hook manager
func NewManager() *Manager {
	return &Manager{
		hooks:  make(map[EventType][]hookEntry),
		logger: &defaultLogger{},
	}
}

// NewManagerWithLogger creates a new hook manager with a custom logger
func NewManagerWithLogger(logger Logger) *Manager {
	if logger == nil {
		logger = &defaultLogger{}
	}
	return &Manager{
		hooks:  make(map[EventType][]hookEntry),
		logger: logger,
	}
}

// SetLogger sets the logger for the manager
func (m *Manager) SetLogger(logger Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if logger != nil {
		m.logger = logger
	}
}

// Register registers a hook for its declared events
func (m *Manager) Register(h Hook) error {
	if h == nil {
		return fmt.Errorf("cannot register nil hook")
	}

	events := h.Events()
	if len(events) == 0 {
		return fmt.Errorf("hook %q declares no events", h.Name())
	}

	// Determine priority
	priority := PriorityNormal
	if ph, ok := h.(PriorityHook); ok {
		priority = ph.Priority()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range events {
		// Check for duplicate registration
		for _, entry := range m.hooks[event] {
			if entry.hook.Name() == h.Name() {
				return fmt.Errorf("hook %q already registered for event %q", h.Name(), event)
			}
		}

		m.hooks[event] = append(m.hooks[event], hookEntry{
			hook:     h,
			priority: priority,
		})

		// Re-sort by priority (descending)
		sort.Slice(m.hooks[event], func(i, j int) bool {
			return m.hooks[event][i].priority > m.hooks[event][j].priority
		})
	}

	m.logger.Printf("[hooks] Registered hook %q for events: %v", h.Name(), events)
	return nil
}

// Unregister removes a hook by name
func (m *Manager) Unregister(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := false
	for event, entries := range m.hooks {
		filtered := make([]hookEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.hook.Name() != name {
				filtered = append(filtered, entry)
			} else {
				removed = true
			}
		}
		m.hooks[event] = filtered
	}

	if removed {
		m.logger.Printf("[hooks] Unregistered hook %q", name)
	}
	return removed
}

// Execute runs all hooks registered for the given event type
// Hooks are executed in priority order (highest first)
// Errors are logged but do not stop execution of subsequent hooks
func (m *Manager) Execute(ctx context.Context, event EventType, hookCtx *HookContext) []error {
	m.mu.RLock()
	entries := make([]hookEntry, len(m.hooks[event]))
	copy(entries, m.hooks[event])
	m.mu.RUnlock()

	if len(entries) == 0 {
		return nil
	}

	m.logger.Printf("[hooks] Executing %d hook(s) for event %q", len(entries), event)

	var errors []error
	for _, entry := range entries {
		h := entry.hook

		// Check conditional execution
		if ch, ok := h.(ConditionalHook); ok {
			if !ch.ShouldExecute(ctx, event, hookCtx) {
				m.logger.Printf("[hooks] Skipping hook %q (condition not met)", h.Name())
				continue
			}
		}

		// Execute the hook
		m.logger.Printf("[hooks] Running hook %q", h.Name())
		if err := h.Execute(ctx, event, hookCtx); err != nil {
			m.logger.Printf("[hooks] Hook %q failed: %v", h.Name(), err)
			errors = append(errors, fmt.Errorf("hook %q: %w", h.Name(), err))
		}
	}

	return errors
}

// ExecuteAsync runs all async-capable hooks concurrently
// Non-async hooks are still run sequentially
func (m *Manager) ExecuteAsync(ctx context.Context, event EventType, hookCtx *HookContext) []error {
	m.mu.RLock()
	entries := make([]hookEntry, len(m.hooks[event]))
	copy(entries, m.hooks[event])
	m.mu.RUnlock()

	if len(entries) == 0 {
		return nil
	}

	// Separate async and sync hooks
	var asyncHooks []hookEntry
	var syncHooks []hookEntry

	for _, entry := range entries {
		if ah, ok := entry.hook.(AsyncHook); ok && ah.IsAsync() {
			asyncHooks = append(asyncHooks, entry)
		} else {
			syncHooks = append(syncHooks, entry)
		}
	}

	var errors []error
	var errMu sync.Mutex

	// Run async hooks concurrently
	var wg sync.WaitGroup
	for _, entry := range asyncHooks {
		wg.Add(1)
		go func(h Hook) {
			defer wg.Done()

			// Check conditional execution
			if ch, ok := h.(ConditionalHook); ok {
				if !ch.ShouldExecute(ctx, event, hookCtx) {
					return
				}
			}

			if err := h.Execute(ctx, event, hookCtx); err != nil {
				errMu.Lock()
				errors = append(errors, fmt.Errorf("hook %q: %w", h.Name(), err))
				errMu.Unlock()
			}
		}(entry.hook)
	}

	// Run sync hooks sequentially
	for _, entry := range syncHooks {
		h := entry.hook

		// Check conditional execution
		if ch, ok := h.(ConditionalHook); ok {
			if !ch.ShouldExecute(ctx, event, hookCtx) {
				continue
			}
		}

		if err := h.Execute(ctx, event, hookCtx); err != nil {
			errors = append(errors, fmt.Errorf("hook %q: %w", h.Name(), err))
		}
	}

	wg.Wait()
	return errors
}

// HasHooksFor returns true if any hooks are registered for the event
func (m *Manager) HasHooksFor(event EventType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.hooks[event]) > 0
}

// GetHooksFor returns the names of hooks registered for the event
func (m *Manager) GetHooksFor(event EventType) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := m.hooks[event]
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.hook.Name()
	}
	return names
}

// ListAllHooks returns all registered hooks grouped by event
func (m *Manager) ListAllHooks() map[EventType][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[EventType][]string)
	for event, entries := range m.hooks {
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.hook.Name()
		}
		result[event] = names
	}
	return result
}

// HookCount returns the total number of registered hooks
func (m *Manager) HookCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Count unique hooks
	seen := make(map[string]bool)
	for _, entries := range m.hooks {
		for _, entry := range entries {
			seen[entry.hook.Name()] = true
		}
	}
	return len(seen)
}

// Clear removes all registered hooks
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks = make(map[EventType][]hookEntry)
	m.logger.Printf("[hooks] Cleared all hooks")
}
