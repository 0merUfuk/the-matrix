package matrixcfg

import "testing"

func TestIntOr(t *testing.T) {
	t.Run("nil returns fallback", func(t *testing.T) {
		got := IntOr(nil, 42)
		if got != 42 {
			t.Errorf("expected 42, got %d", got)
		}
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		v := 99
		got := IntOr(&v, 42)
		if got != 99 {
			t.Errorf("expected 99, got %d", got)
		}
	})

	t.Run("zero value is valid", func(t *testing.T) {
		v := 0
		got := IntOr(&v, 42)
		if got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})
}

func TestFloat64Or(t *testing.T) {
	t.Run("nil returns fallback", func(t *testing.T) {
		got := Float64Or(nil, 3.14)
		if got != 3.14 {
			t.Errorf("expected 3.14, got %g", got)
		}
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		v := 2.71
		got := Float64Or(&v, 3.14)
		if got != 2.71 {
			t.Errorf("expected 2.71, got %g", got)
		}
	})

	t.Run("zero value is valid", func(t *testing.T) {
		v := 0.0
		got := Float64Or(&v, 3.14)
		if got != 0.0 {
			t.Errorf("expected 0.0, got %g", got)
		}
	})
}

func TestStringOr(t *testing.T) {
	t.Run("nil returns fallback", func(t *testing.T) {
		got := StringOr(nil, "default")
		if got != "default" {
			t.Errorf("expected %q, got %q", "default", got)
		}
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		v := "custom"
		got := StringOr(&v, "default")
		if got != "custom" {
			t.Errorf("expected %q, got %q", "custom", got)
		}
	})

	t.Run("empty string is valid", func(t *testing.T) {
		v := ""
		got := StringOr(&v, "default")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}

func TestBoolOr(t *testing.T) {
	t.Run("nil returns fallback true", func(t *testing.T) {
		got := BoolOr(nil, true)
		if got != true {
			t.Errorf("expected true, got %v", got)
		}
	})

	t.Run("nil returns fallback false", func(t *testing.T) {
		got := BoolOr(nil, false)
		if got != false {
			t.Errorf("expected false, got %v", got)
		}
	})

	t.Run("non-nil true returns true", func(t *testing.T) {
		v := true
		got := BoolOr(&v, false)
		if got != true {
			t.Errorf("expected true, got %v", got)
		}
	})

	t.Run("false is valid value not unset", func(t *testing.T) {
		v := false
		got := BoolOr(&v, true)
		if got != false {
			t.Errorf("expected false (explicit value), got %v", got)
		}
	})
}
