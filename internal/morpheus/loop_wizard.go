package morpheus

import (
	"fmt"
	"strconv"
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
	mcStr, err := wizard.AskInput("Max cycles:", strconv.Itoa(cc.MaxCycles), nil)
	if err != nil {
		return "", CycleConfig{}, err
	}
	cc.MaxCycles, _ = strconv.Atoi(strings.TrimSpace(mcStr))
	if cc.MaxCycles <= 0 {
		cc.MaxCycles = defaultConfig.MaxCycles
	}

	dtStr, err := wizard.AskInput("Developer max turns:", strconv.Itoa(cc.DeveloperMaxTurns), nil)
	if err != nil {
		return "", CycleConfig{}, err
	}
	cc.DeveloperMaxTurns, _ = strconv.Atoi(strings.TrimSpace(dtStr))
	if cc.DeveloperMaxTurns <= 0 {
		cc.DeveloperMaxTurns = defaultConfig.DeveloperMaxTurns
	}

	ttStr, err := wizard.AskInput("Tester max turns:", strconv.Itoa(cc.TesterMaxTurns), nil)
	if err != nil {
		return "", CycleConfig{}, err
	}
	cc.TesterMaxTurns, _ = strconv.Atoi(strings.TrimSpace(ttStr))
	if cc.TesterMaxTurns <= 0 {
		cc.TesterMaxTurns = defaultConfig.TesterMaxTurns
	}

	rtStr, err := wizard.AskInput("Reviewer max turns:", strconv.Itoa(cc.ReviewerMaxTurns), nil)
	if err != nil {
		return "", CycleConfig{}, err
	}
	cc.ReviewerMaxTurns, _ = strconv.Atoi(strings.TrimSpace(rtStr))
	if cc.ReviewerMaxTurns <= 0 {
		cc.ReviewerMaxTurns = defaultConfig.ReviewerMaxTurns
	}

	return goal, cc, nil
}
