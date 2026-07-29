package morpheus

import (
	"strconv"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// askCycleValue reads one cycle setting and preserves the supplied default for
// invalid or non-positive input.
func askCycleValue(prompt string, fallback int) (int, error) {
	input, err := wizard.AskInput(prompt, strconv.Itoa(fallback), nil)
	if err != nil {
		return 0, err
	}

	value, _ := strconv.Atoi(strings.TrimSpace(input))
	if value <= 0 {
		return fallback, nil
	}
	return value, nil
}
