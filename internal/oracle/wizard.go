package oracle

import (
	"fmt"
	"time"

	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// RunStackWizard runs the 10-question interactive wizard and returns a StackContext.
func RunStackWizard() (StackContext, error) {
	var ctx StackContext

	fmt.Println()

	// Q1: Stack name (kebab-case)
	stackName, err := wizard.AskInput("Stack name (kebab-case, e.g. nextjs-typescript):", "", wizard.ValidateKebabCase)
	if err != nil {
		return ctx, err
	}
	ctx.StackName = stackName

	// Q2: Framework version
	frameworkVersion, err := wizard.AskInput("Framework / language version (e.g. \"Next.js 15 / TypeScript 5.x\"):", "", wizard.ValidateMinLength(1))
	if err != nil {
		return ctx, err
	}
	ctx.FrameworkVersion = frameworkVersion

	// Q3: App type
	appType, err := wizard.AskSelect("Application type:", []wizard.Option{
		{Label: "Full-stack (frontend + backend in one codebase)", Value: "full-stack"},
		{Label: "API-only (backend service / microservice)", Value: "api-only"},
		{Label: "Frontend (SPA or SSR, no server-side API)", Value: "frontend"},
		{Label: "CLI tool", Value: "cli"},
		{Label: "Library / SDK", Value: "library"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.AppType = appType

	// Q4: Architecture style
	archStyle, err := wizard.AskSelect("Architecture style:", []wizard.Option{
		{Label: "Layered (Handler → Service → Repository)", Value: "layered"},
		{Label: "Feature-based (co-located by domain feature)", Value: "feature-based"},
		{Label: "Flat (small app, minimal structure)", Value: "flat"},
		{Label: "Hybrid (layered core + feature modules)", Value: "hybrid"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.ArchStyle = archStyle

	// Q5: State management
	stateApproach, err := wizard.AskSelect("State management approach:", []wizard.Option{
		{Label: "Server-side only (SSR / RSC, no client state library)", Value: "server-side"},
		{Label: "Client-state (React Query, Zustand, Redux, etc.)", Value: "client-state"},
		{Label: "Hybrid (RSC/SSR server + light client state)", Value: "hybrid"},
		{Label: "N/A (API-only, CLI, or library)", Value: "n/a"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.StateApproach = stateApproach

	// Q6: Data layer
	dataLayer, err := wizard.AskInput("Data layer / ORM (e.g. \"Prisma\", \"Drizzle\", \"SQLAlchemy\", \"none\" for API-only):", "none", nil)
	if err != nil {
		return ctx, err
	}
	ctx.DataLayer = dataLayer

	// Q7: Testing framework
	testingFramework, err := wizard.AskInput("Testing framework(s) (e.g. \"Jest + Testing Library + Playwright\"):", "Jest", nil)
	if err != nil {
		return ctx, err
	}
	ctx.TestingFramework = testingFramework

	// Q8a: Queue
	hasQueue, err := wizard.AskConfirm("Does this stack typically use a message queue (RabbitMQ, SQS, BullMQ, etc.)?", false)
	if err != nil {
		return ctx, err
	}
	ctx.HasQueue = hasQueue

	// Q8b: Background jobs (only if queue = yes)
	if hasQueue {
		hasBackgroundJobs, err := wizard.AskConfirm("Does it also use background job workers?", true)
		if err != nil {
			return ctx, err
		}
		ctx.HasBackgroundJobs = hasBackgroundJobs
	}

	// Q9: Topics
	acceptAllTopics, err := wizard.AskConfirm(fmt.Sprintf("Include all %d standard topics?", len(AllTopics)), true)
	if err != nil {
		return ctx, err
	}

	if acceptAllTopics {
		ctx.SelectedTopics = AllTopicValues()
	} else {
		// Build checkbox options
		var options []wizard.Option
		for _, t := range AllTopics {
			options = append(options, wizard.Option{Label: t.Name, Value: t.Value})
		}
		selected, err := wizard.AskCheckbox("Select topics to include (space to toggle):", options)
		if err != nil {
			return ctx, err
		}
		if len(selected) == 0 {
			fmt.Println("  No topics selected — defaulting to all.")
			ctx.SelectedTopics = AllTopicValues()
		} else {
			ctx.SelectedTopics = selected
		}
	}

	// Q10: Output path
	defaultOutput := fmt.Sprintf("/tmp/oracle-output/%s", ctx.StackName)
	outputPath, err := wizard.AskInput("Output directory:", defaultOutput, nil)
	if err != nil {
		return ctx, err
	}
	if outputPath == "" {
		outputPath = defaultOutput
	}
	ctx.OutputPath = outputPath

	// Computed fields
	ctx.TopicCount = len(ctx.SelectedTopics)
	ctx.CreatedDate = time.Now().Format("2006-01-02")

	return ctx, nil
}
