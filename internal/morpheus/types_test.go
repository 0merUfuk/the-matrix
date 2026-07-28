package morpheus

import (
	"testing"
)

func TestCalculateCycles_Low(t *testing.T) {
	cfg := CalculateCycles([]string{"User", "Order"}, nil)
	if cfg.MaxCycles != 12 {
		t.Errorf("expected maxCycles=12, got %d", cfg.MaxCycles)
	}
	if cfg.DeveloperMaxTurns != 28 {
		t.Errorf("expected devTurns=28, got %d", cfg.DeveloperMaxTurns)
	}
}

func TestCalculateCycles_Medium(t *testing.T) {
	cfg := CalculateCycles([]string{"User", "Order", "Payment", "Invoice", "Refund", "Subscription", "Webhook"}, nil)
	if cfg.MaxCycles != 20 {
		t.Errorf("expected maxCycles=20, got %d", cfg.MaxCycles)
	}
}

func TestCalculateCycles_High(t *testing.T) {
	entities := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	patterns := []string{"retry", "hmac", "inbound-webhooks"}
	cfg := CalculateCycles(entities, patterns)
	// complexity = 8 + 2 + 2 + 2 = 14, > 12
	if cfg.MaxCycles != 28 {
		t.Errorf("expected maxCycles=28, got %d", cfg.MaxCycles)
	}
}

func TestDeriveRetryEntity_PreferExecution(t *testing.T) {
	entity := DeriveRetryEntity([]string{"User", "Execution", "Payment"})
	if entity != "Execution" {
		t.Errorf("expected Execution, got %s", entity)
	}
}

func TestDeriveRetryEntity_FallbackToFirst(t *testing.T) {
	entity := DeriveRetryEntity([]string{"Generation", "Credit"})
	if entity != "Generation" {
		t.Errorf("expected Generation (first), got %s", entity)
	}
}

func TestDeriveRetryEntity_Empty(t *testing.T) {
	entity := DeriveRetryEntity(nil)
	if entity != "Entity" {
		t.Errorf("expected Entity (default), got %s", entity)
	}
}

func TestPortRegistry(t *testing.T) {
	// Verify known ports exist
	if KnownPorts["internal-app"] != 3000 {
		t.Error("internal-app should be 3000")
	}
	if KnownPorts["generation-service-api"] != 8082 {
		t.Error("generation-service-api should be 8082")
	}
}

func TestSuggestNextApiPort(t *testing.T) {
	port := SuggestNextApiPort()
	if port < 8086 {
		t.Errorf("expected port >= 8086, got %d", port)
	}
	if port%2 != 0 {
		t.Errorf("API port should be even, got %d", port)
	}
}

func TestIsPortConflict_Conflict(t *testing.T) {
	result := IsPortConflict(3000, "new-service")
	if !result.Conflict {
		t.Error("port 3000 should conflict with internal-app")
	}
	if result.ExistingService != "internal-app" {
		t.Errorf("expected internal-app, got %s", result.ExistingService)
	}
}

func TestIsPortConflict_NoConflict(t *testing.T) {
	result := IsPortConflict(9999, "new-service")
	if result.Conflict {
		t.Error("port 9999 should not conflict")
	}
}

func TestIsPortConflict_SameService(t *testing.T) {
	// Editing an existing service's port should not conflict
	result := IsPortConflict(8082, "generation-service")
	if result.Conflict {
		t.Error("same service should not conflict with itself")
	}
}

func TestValidatePort_Valid(t *testing.T) {
	err := ValidatePort(8090, "test-service", true)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidatePort_OutOfRange(t *testing.T) {
	err := ValidatePort(0, "test-service", true)
	if err == nil {
		t.Error("expected error for port 0")
	}
	err = ValidatePort(70000, "test-service", true)
	if err == nil {
		t.Error("expected error for port 70000")
	}
}
