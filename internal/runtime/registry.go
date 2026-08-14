package runtime

import "sync"

// Registry holds all registered runtimes, keyed by name.
var Registry = &registry{}

type registry struct {
	mu       sync.RWMutex
	runtimes map[string]Runtime
	order    []string
}

// Register adds or replaces a runtime by name.
func (r *registry) Register(rt Runtime) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.runtimes == nil {
		r.runtimes = make(map[string]Runtime)
	}
	if _, exists := r.runtimes[rt.Name()]; !exists {
		r.order = append(r.order, rt.Name())
	}
	r.runtimes[rt.Name()] = rt
}

// Get returns the runtime registered under name, or nil if none exists.
func (r *registry) Get(name string) Runtime {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.runtimes[name]
}

// Default returns Claude when registered, otherwise the first registered runtime.
func (r *registry) Default() Runtime {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if rt, ok := r.runtimes["claude"]; ok {
		return rt
	}
	if len(r.order) > 0 {
		return r.runtimes[r.order[0]]
	}
	return nil
}

// List returns all registered runtimes in registration order.
func (r *registry) List() []Runtime {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Runtime, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.runtimes[name])
	}
	return out
}

// Names returns all registered runtime names in registration order.
func (r *registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]string(nil), r.order...)
}
