package runtime

import "testing"

type mockRuntime struct {
	name string
}

func (m *mockRuntime) Name() string { return m.name }

func (m *mockRuntime) DisplayName() string { return m.name }

func (m *mockRuntime) Capabilities() Capabilities { return Capabilities{} }

func (m *mockRuntime) BuildCommand(InvokeOpts) (*Command, error) { return &Command{}, nil }

func (m *mockRuntime) CheckAuth() AuthStatus { return AuthStatus{} }

func (m *mockRuntime) ContextFiles() ContextFileSpec { return ContextFileSpec{} }

func (m *mockRuntime) AgentFormat() AgentFormat { return AgentFormat{} }

func (m *mockRuntime) SkillFormat() *SkillFormat { return nil }

func (m *mockRuntime) SettingsFormat() SettingsFormat { return SettingsFormat{} }

func (m *mockRuntime) EnvModifications() []EnvModification { return nil }

func (m *mockRuntime) TranslateTools([]string) string { return "" }

func (m *mockRuntime) ResolveModel(ModelTier) string { return "" }

func TestRegistryRegisterAndGet(t *testing.T) {
	r := &registry{}
	want := &mockRuntime{name: "test"}

	r.Register(want)

	if got := r.Get("test"); got != want {
		t.Fatalf("Get(%q) = %v, want %v", "test", got, want)
	}
}

func TestRegistryGetUnregistered(t *testing.T) {
	r := &registry{}

	if got := r.Get("missing"); got != nil {
		t.Fatalf("Get(%q) = %v, want nil", "missing", got)
	}
}

func TestRegistryDefaultReturnsFirstRegistered(t *testing.T) {
	r := &registry{}
	first := &mockRuntime{name: "first"}
	r.Register(first)
	r.Register(&mockRuntime{name: "second"})

	if got := r.Default(); got != first {
		t.Fatalf("Default() = %v, want %v", got, first)
	}
}

func TestRegistryList(t *testing.T) {
	r := &registry{}
	first := &mockRuntime{name: "first"}
	second := &mockRuntime{name: "second"}
	r.Register(first)
	r.Register(second)

	got := r.List()
	if len(got) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(got))
	}
	if got[0] != first || got[1] != second {
		t.Fatalf("List() = %v, want [%v %v]", got, first, second)
	}
}

func TestRegistryNames(t *testing.T) {
	r := &registry{}
	r.Register(&mockRuntime{name: "first"})
	r.Register(&mockRuntime{name: "second"})

	got := r.Names()
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("Names() = %v, want [first second]", got)
	}
}

func TestRegistryRegisterOverwritesSameName(t *testing.T) {
	r := &registry{}
	first := &mockRuntime{name: "same"}
	replacement := &mockRuntime{name: "same"}
	r.Register(first)
	r.Register(replacement)

	if got := r.Get("same"); got != replacement {
		t.Fatalf("Get(%q) = %v, want replacement %v", "same", got, replacement)
	}
	if got := len(r.List()); got != 1 {
		t.Fatalf("len(List()) = %d, want 1", got)
	}
}
