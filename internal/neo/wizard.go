package neo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// RunProjectWizard runs the 12-question adaptive wizard and returns a ProjectProfile.
func RunProjectWizard() (ProjectProfile, error) {
	var profile ProjectProfile
	profile.CreatedDate = time.Now().Format("2006-01-02")

	fmt.Println()

	// Q1: Project name
	name, err := wizard.AskInput("Project name (kebab-case):", "", wizard.ValidateKebabCase)
	if err != nil {
		return profile, err
	}
	profile.ProjectName = name

	// Q2: Project type
	projectType, err := wizard.AskSelect("Project type:", []wizard.Option{
		{Label: "Single application", Value: "single-app"},
		{Label: "Monorepo (multiple apps in one repo)", Value: "monorepo"},
		{Label: "Multi-repo microservices", Value: "multi-repo-microservices"},
	})
	if err != nil {
		return profile, err
	}
	profile.ProjectType = projectType

	// Q3: Primary stack
	stack, err := collectStack("Primary stack — ")
	if err != nil {
		return profile, err
	}
	profile.Stacks = append(profile.Stacks, stack)

	// Q8: Additional stacks?
	for {
		addMore, err := wizard.AskConfirm("Add another stack?", false)
		if err != nil {
			return profile, err
		}
		if !addMore {
			break
		}
		extra, err := collectStack("Additional stack — ")
		if err != nil {
			return profile, err
		}
		profile.Stacks = append(profile.Stacks, extra)
	}

	// Q9: Team size
	teamSize, err := wizard.AskSelect("Team size:", []wizard.Option{
		{Label: "Solo (1 developer)", Value: "solo"},
		{Label: "Small team (2-5 developers)", Value: "small"},
		{Label: "Mid team (5-20 developers)", Value: "mid"},
	})
	if err != nil {
		return profile, err
	}
	profile.TeamSize = teamSize

	// Q10: Services (conditional on multi-repo)
	if projectType == "multi-repo-microservices" {
		countStr, err := wizard.AskInput("How many services?", "1", nil)
		if err != nil {
			return profile, err
		}
		count, _ := strconv.Atoi(strings.TrimSpace(countStr))
		if count < 1 {
			count = 1
		}

		for i := 0; i < count; i++ {
			svcName, err := wizard.AskInput(fmt.Sprintf("Service %d name:", i+1), "", wizard.ValidateKebabCase)
			if err != nil {
				return profile, err
			}

			// Pick stack from available stacks
			var stackName string
			if len(profile.Stacks) > 0 {
				stackName = profile.Stacks[0].Name
			}
			if len(profile.Stacks) > 1 {
				var opts []wizard.Option
				for _, s := range profile.Stacks {
					opts = append(opts, wizard.Option{Label: s.Name + " (" + s.Language + ")", Value: s.Name})
				}
				stackName, err = wizard.AskSelect(fmt.Sprintf("Stack for %s:", svcName), opts)
				if err != nil {
					return profile, err
				}
			}

			portStr, err := wizard.AskInput(fmt.Sprintf("API port for %s:", svcName), strconv.Itoa(8080+i*2), nil)
			if err != nil {
				return profile, err
			}
			port, _ := strconv.Atoi(strings.TrimSpace(portStr))

			profile.Services = append(profile.Services, ServiceProfile{
				Name:  svcName,
				Stack: stackName,
				Port:  port,
			})
		}
	}

	// Q11: originating project auto-detection
	cwd, _ := os.Getwd()
	claudeMdPath := filepath.Join(cwd, "CLAUDE.md")
	autoDetected := false
	if config.FileExists(claudeMdPath) {
		content, err := config.ReadFileString(claudeMdPath)
		if err == nil && strings.Contains(content, "originating project") {
			autoDetected = true
		}
	}

	if autoDetected {
		fmt.Println("  originating project project detected (CLAUDE.md contains originating project marker)")
		profile.IsInternal = true
	} else {
		isInternal, err := wizard.AskConfirm("Is this a originating project project?", false)
		if err != nil {
			return profile, err
		}
		profile.IsInternal = isInternal
	}

	// Q12: Output directory
	defaultOutput, _ := os.Getwd()
	outputPath, err := wizard.AskInput("Output directory:", defaultOutput, nil)
	if err != nil {
		return profile, err
	}
	if outputPath == "" {
		outputPath = defaultOutput
	}
	profile.OutputPath, _ = filepath.Abs(outputPath)

	return profile, nil
}

// collectStack collects a single StackProfile via prompts.
func collectStack(prefix string) (StackProfile, error) {
	var s StackProfile

	// Stack name
	name, err := wizard.AskInput(prefix+"Stack name (kebab-case, e.g. go-api):", "", wizard.ValidateKebabCase)
	if err != nil {
		return s, err
	}
	s.Name = name

	// Language
	lang, err := wizard.AskSelect(prefix+"Language:", []wizard.Option{
		{Label: "Go", Value: "go"},
		{Label: "Node.js / TypeScript", Value: "nodejs"},
		{Label: "Python", Value: "python"},
		{Label: "Flutter / Dart", Value: "dart"},
		{Label: "Other", Value: "other"},
	})
	if err != nil {
		return s, err
	}
	s.Language = lang

	// Framework
	framework, err := wizard.AskInput(prefix+"Framework (e.g. \"chi v5\", \"Next.js 15\", \"FastAPI\"):", "", nil)
	if err != nil {
		return s, err
	}
	s.Framework = framework

	// Architecture style
	archStyle, err := wizard.AskSelect(prefix+"Architecture style:", []wizard.Option{
		{Label: "Layered (Handler → Service → Repository)", Value: "layered"},
		{Label: "Feature-based (co-located by domain)", Value: "feature-based"},
		{Label: "Flat (minimal structure)", Value: "flat"},
		{Label: "Hybrid", Value: "hybrid"},
	})
	if err != nil {
		return s, err
	}
	s.ArchStyle = archStyle

	// Testing framework
	testFramework, err := wizard.AskInput(prefix+"Testing framework:", "Jest", nil)
	if err != nil {
		return s, err
	}
	s.TestingFramework = testFramework

	// Data layer
	dataLayer, err := wizard.AskInput(prefix+"Data layer / ORM (or \"none\"):", "none", nil)
	if err != nil {
		return s, err
	}
	s.DataLayer = dataLayer

	// Queue
	hasQueue, err := wizard.AskConfirm(prefix+"Uses a message queue?", false)
	if err != nil {
		return s, err
	}
	s.HasQueue = hasQueue

	return s, nil
}
