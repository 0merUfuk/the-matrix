package wizard

import (
	"testing"
)

func TestValidateKebabCase(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"my-service", true},
		{"cron-service", true},
		{"a", true},
		{"a1", true},
		{"my-service-2", true},
		{"", false},
		{"MyService", false},
		{"my_service", false},
		{"1service", false},
		{"-service", false},
		{"my service", false},
		{"MY-SERVICE", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateKebabCase(tt.input)
			if (err == nil) != tt.valid {
				t.Errorf("ValidateKebabCase(%q) error = %v, want valid = %v", tt.input, err, tt.valid)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"8080", true},
		{"1", true},
		{"65535", true},
		{"0", false},
		{"65536", false},
		{"-1", false},
		{"abc", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidatePort(tt.input)
			if (err == nil) != tt.valid {
				t.Errorf("ValidatePort(%q) error = %v, want valid = %v", tt.input, err, tt.valid)
			}
		})
	}
}

func TestValidateMinLength(t *testing.T) {
	validate := ValidateMinLength(5)
	if err := validate("hello"); err != nil {
		t.Errorf("5 chars should pass min length 5, got: %v", err)
	}
	if err := validate("hi"); err == nil {
		t.Error("2 chars should fail min length 5")
	}
	if err := validate("  hi  "); err == nil {
		t.Error("2 chars (trimmed) should fail min length 5")
	}
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a, b, c", 3},
		{"User,Order,Payment", 3},
		{" , , ", 0},
		{"", 0},
		{"single", 1},
		{" spaced , items , here ", 3},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseCommaSeparated(tt.input)
			if len(got) != tt.want {
				t.Errorf("ParseCommaSeparated(%q) = %d items, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-service", "MyService"},
		{"cron_service", "CronService"},
		{"hello world", "HelloWorld"},
		{"simple", "Simple"},
		{"already-PascalCase", "AlreadyPascalCase"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTodayISO(t *testing.T) {
	result := TodayISO()
	if len(result) != 10 {
		t.Errorf("TodayISO() = %q, want YYYY-MM-DD format", result)
	}
}
