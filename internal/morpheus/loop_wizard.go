package morpheus

import (
	"fmt"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// RunLoopWizard asks 2 questions: goal (optional) and cycle config.
func RunLoopWizard() (goal string, cc CycleConfig, err error) {
	// Q1: Goal (optional)
	goal, err = wizard.AskInput(
		"What should the loop accomplish? (empty = manual tasks/todo.md):",
		"",
		nil,
	)
	if err != nil {
		return "", CycleConfig{}, err
	}
	goal = strings.TrimSpace(goal)

	// Q2: Cycle config — show defaults, offer override
	defaultConfig := defaultCycleConfig

	fmt.Printf("\n  Default cycle config: max=%d, dev=%d, test=%d, review=%d\n",
		defaultConfig.MaxCycles, defaultConfig.DeveloperMaxTurns,
		defaultConfig.TesterMaxTurns, defaultConfig.ReviewerMaxTurns)

	acceptDefaults, err := wizard.AskConfirm("Accept these cycle settings?", true)
	if err != nil {
		return "", CycleConfig{}, err
	}

	if acceptDefaults {
		return goal, defaultConfig, nil
	}

	// Manual override
	cc = defaultConfig
	cc.MaxCycles, err = askCycleValue("Max cycles:", cc.MaxCycles)
	if err != nil {
		return "", CycleConfig{}, err
	}

	cc.DeveloperMaxTurns, err = askCycleValue("Developer max turns:", cc.DeveloperMaxTurns)
	if err != nil {
		return "", CycleConfig{}, err
	}

	cc.TesterMaxTurns, err = askCycleValue("Tester max turns:", cc.TesterMaxTurns)
	if err != nil {
		return "", CycleConfig{}, err
	}

	cc.ReviewerMaxTurns, err = askCycleValue("Reviewer max turns:", cc.ReviewerMaxTurns)
	if err != nil {
		return "", CycleConfig{}, err
	}

	return goal, cc, nil
}
