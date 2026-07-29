package morpheus

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// RunEntryWizard prompts for project type and mode.
func RunEntryWizard() (projectType, mode string, err error) {
	projectType, err = wizard.AskSelect("Project type:", []wizard.Option{
		{Label: "Internal Microservice", Value: "internal"},
		{Label: "General Project (any stack)", Value: "general"},
	})
	if err != nil {
		return "", "", err
	}

	mode, err = wizard.AskSelect("Scaffolding mode:", []wizard.Option{
		{Label: "From scratch — new service, full scaffold", Value: "from-scratch"},
		{Label: "Post-v1 — add .claude/ to existing service", Value: "post-v1"},
	})
	if err != nil {
		return "", "", err
	}

	return projectType, mode, nil
}

// RunInternalWizard runs the 14-question originating project from-scratch wizard.
func RunInternalWizard() (ProjectContext, error) {
	var ctx ProjectContext
	ctx.ProjectType = "internal"
	ctx.IsInternal = true
	ctx.CreatedDate = time.Now().Format("2006-01-02")

	// Q1: Service name
	name, err := wizard.AskInput("Service name (kebab-case):", "", wizard.ValidateKebabCase)
	if err != nil {
		return ctx, err
	}
	ctx.ServiceName = name
	ctx.ServiceNamePascal = wizard.ToPascalCase(name)

	// Q2: Tech stack
	tech, err := wizard.AskSelect("Tech stack:", []wizard.Option{
		{Label: "Go 1.24 (chi, zerolog, bun, ginkgo)", Value: "go"},
		{Label: "Node.js 20+ (Express, Prisma, Jest)", Value: "node"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.Tech = tech
	ctx.IsGo = tech == "go"
	ctx.IsNode = tech == "node"
	if ctx.IsGo {
		ctx.GoVersion = "1.24"
	}
	if ctx.IsNode {
		ctx.NodeArchitecture = "4-layer"
	}

	// Q3: Migration?
	isMigration, err := wizard.AskConfirm("Is this migrating functionality from internal-app?", false)
	if err != nil {
		return ctx, err
	}
	ctx.IsMigration = isMigration

	// Q4: Migration source (conditional)
	if isMigration {
		source, err := wizard.AskInput("Migration source module (e.g. internal-app/modules/generation):", "", nil)
		if err != nil {
			return ctx, err
		}
		ctx.MigrationSource = source
	}

	// Q5: Description
	desc, err := wizard.AskInput("Service description (1-3 sentences):", "", wizard.ValidateMinLength(5))
	if err != nil {
		return ctx, err
	}
	ctx.Description = desc

	// Q6: API port
	defaultPort := SuggestNextApiPort()
	portStr, err := wizard.AskInput(fmt.Sprintf("API port (default: %d):", defaultPort), strconv.Itoa(defaultPort), func(input string) error {
		p, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		return ValidatePort(p, ctx.ServiceName, true)
	})
	if err != nil {
		return ctx, err
	}
	ctx.ApiPort, _ = strconv.Atoi(strings.TrimSpace(portStr))

	// Q7: Worker?
	hasWorker, err := wizard.AskConfirm("Does this service need a background worker?", false)
	if err != nil {
		return ctx, err
	}
	ctx.HasWorker = hasWorker

	// Q8: Worker port (conditional)
	if hasWorker {
		defaultWorkerPort := ctx.ApiPort + 1
		wpStr, err := wizard.AskInput(fmt.Sprintf("Worker port (default: %d):", defaultWorkerPort), strconv.Itoa(defaultWorkerPort), nil)
		if err != nil {
			return ctx, err
		}
		ctx.WorkerPort, _ = strconv.Atoi(strings.TrimSpace(wpStr))
	}

	// Q9: Domain entities
	entitiesStr, err := wizard.AskInput("Domain entities (PascalCase, comma-separated, e.g. \"User, Order, Payment\"):", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.Entities = wizard.ParseCommaSeparated(entitiesStr)
	ctx.EntityCount = len(ctx.Entities)

	// Q10: Core operations
	ops, err := wizard.AskInput("Core operations (describe main functionality):", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.CoreOperations = ops

	// Q11: Data stores
	dataStores, err := wizard.AskCheckbox("Data stores:", []wizard.Option{
		{Label: "PostgreSQL", Value: "postgresql"},
		{Label: "Redis", Value: "redis"},
		{Label: "RabbitMQ", Value: "rabbitmq"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.DataStores = dataStores
	for _, ds := range dataStores {
		switch ds {
		case "postgresql":
			ctx.HasPostgres = true
		case "redis":
			ctx.HasRedis = true
		case "rabbitmq":
			ctx.HasRabbitMQ = true
		}
	}

	// Q11b: RabbitMQ details (conditional)
	if ctx.HasRabbitMQ {
		exchange, err := wizard.AskInput("RabbitMQ exchange name:", ctx.ServiceName, nil)
		if err != nil {
			return ctx, err
		}
		ctx.ExchangeName = exchange

		defaultQueues := fmt.Sprintf("%s.deliveries, %s.retry, %s.dlq", exchange, exchange, exchange)
		queuesStr, err := wizard.AskInput("Queue names (comma-separated):", defaultQueues, nil)
		if err != nil {
			return ctx, err
		}
		ctx.Queues = wizard.ParseCommaSeparated(queuesStr)
	}

	// Q12: Auth requirements
	auth, err := wizard.AskCheckbox("Auth requirements:", []wizard.Option{
		{Label: "JWT (Bearer token)", Value: "jwt"},
		{Label: "X-Internal-API-Key", Value: "internal-key"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.AuthRequirements = auth
	for _, a := range auth {
		switch a {
		case "jwt":
			ctx.HasJWT = true
		case "internal-key":
			ctx.HasInternalKey = true
		}
	}

	// Q13: Special patterns
	patterns, err := wizard.AskCheckbox("Special patterns:", []wizard.Option{
		{Label: "HMAC signatures", Value: "hmac"},
		{Label: "Retry with exponential backoff", Value: "retry"},
		{Label: "Inbound webhooks", Value: "inbound-webhooks"},
		{Label: "Rate limiting", Value: "rate-limiting"},
		{Label: "Redis caching", Value: "caching"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.SpecialPatterns = patterns
	for _, p := range patterns {
		switch p {
		case "hmac":
			ctx.HasHMAC = true
		case "retry":
			ctx.HasRetry = true
		case "inbound-webhooks":
			ctx.HasInboundWebhooks = true
		case "rate-limiting":
			ctx.HasRateLimit = true
		case "caching":
			ctx.HasCaching = true
		}
	}

	// Q13b: Dead letter mechanism (conditional on retry)
	if ctx.HasRetry {
		if ctx.HasRabbitMQ {
			dlm, err := wizard.AskSelect("Dead-letter mechanism:", []wizard.Option{
				{Label: "RabbitMQ DLQ", Value: "rabbitmq-dlq"},
				{Label: "PostgreSQL status field", Value: "postgresql-status-field"},
			})
			if err != nil {
				return ctx, err
			}
			ctx.DeadLetterMechanism = dlm
		} else {
			ctx.DeadLetterMechanism = "postgresql-status-field"
		}
		ctx.RetryEntity = DeriveRetryEntity(ctx.Entities)
		ctx.RetryAttemptField = "attempt"
	}

	// Q14: External dependencies
	depsStr, err := wizard.AskInput("External originating project service dependencies (comma-separated, or empty):", "", nil)
	if err != nil {
		return ctx, err
	}
	if strings.TrimSpace(depsStr) != "" {
		ctx.ExternalDependencies = wizard.ParseCommaSeparated(depsStr)
	}

	// Q15: Module path
	defaultModule := fmt.Sprintf("github.com/originating-project/%s", ctx.ServiceName)
	if ctx.IsNode {
		defaultModule = fmt.Sprintf("@originating-project/%s", ctx.ServiceName)
	}
	modulePath, err := wizard.AskInput("Module path:", defaultModule, nil)
	if err != nil {
		return ctx, err
	}
	ctx.ModulePath = normalizeModulePath(modulePath)

	// Q16: Cycle config
	cycleConfig := CalculateCycles(ctx.Entities, ctx.SpecialPatterns)
	fmt.Printf("\n  Auto-calculated cycles: max=%d, dev=%d, test=%d, review=%d\n",
		cycleConfig.MaxCycles, cycleConfig.DeveloperMaxTurns, cycleConfig.TesterMaxTurns, cycleConfig.ReviewerMaxTurns)

	acceptCycles, err := wizard.AskConfirm("Accept these cycle settings?", true)
	if err != nil {
		return ctx, err
	}
	if acceptCycles {
		ctx.MaxCycles = cycleConfig.MaxCycles
		ctx.DeveloperMaxTurns = cycleConfig.DeveloperMaxTurns
		ctx.TesterMaxTurns = cycleConfig.TesterMaxTurns
		ctx.ReviewerMaxTurns = cycleConfig.ReviewerMaxTurns
	} else {
		// Manual override — propagate errors and guard against non-positive
		// values so loop.sh is never rendered with zeros.
		ctx.MaxCycles, err = askCycleValue("Max cycles:", cycleConfig.MaxCycles)
		if err != nil {
			return ctx, fmt.Errorf("reading max cycles: %w", err)
		}

		ctx.DeveloperMaxTurns, err = askCycleValue("Developer max turns:", cycleConfig.DeveloperMaxTurns)
		if err != nil {
			return ctx, fmt.Errorf("reading developer max turns: %w", err)
		}

		ctx.TesterMaxTurns, err = askCycleValue("Tester max turns:", cycleConfig.TesterMaxTurns)
		if err != nil {
			return ctx, fmt.Errorf("reading tester max turns: %w", err)
		}

		ctx.ReviewerMaxTurns, err = askCycleValue("Reviewer max turns:", cycleConfig.ReviewerMaxTurns)
		if err != nil {
			return ctx, fmt.Errorf("reading reviewer max turns: %w", err)
		}
	}

	return ctx, nil
}

// RunInternalWizardWithContext runs the originating project wizard with pre-filled identity/tech fields.
// Skips Q1 (service name) and Q2 (tech stack), asks everything else.
func RunInternalWizardWithContext(ctx ProjectContext) (ProjectContext, error) {
	// Set GoVersion default if Go project (Tech was pre-filled by caller)
	if ctx.IsGo && ctx.GoVersion == "" {
		ctx.GoVersion = "1.24"
	}

	// Q3: Migration?
	isMigration, err := wizard.AskConfirm("Is this migrating functionality from internal-app?", false)
	if err != nil {
		return ctx, err
	}
	ctx.IsMigration = isMigration

	// Q4: Migration source (conditional)
	if isMigration {
		source, err := wizard.AskInput("Migration source module (e.g. internal-app/modules/generation):", "", nil)
		if err != nil {
			return ctx, err
		}
		ctx.MigrationSource = source
	}

	// Q5: Description
	desc, err := wizard.AskInput("Service description (1-3 sentences):", "", wizard.ValidateMinLength(5))
	if err != nil {
		return ctx, err
	}
	ctx.Description = desc

	// Q6: API port
	defaultPort := SuggestNextApiPort()
	portStr, err := wizard.AskInput(fmt.Sprintf("API port (default: %d):", defaultPort), strconv.Itoa(defaultPort), func(input string) error {
		p, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		return ValidatePort(p, ctx.ServiceName, true)
	})
	if err != nil {
		return ctx, err
	}
	ctx.ApiPort, _ = strconv.Atoi(strings.TrimSpace(portStr))

	// Q7: Worker?
	hasWorker, err := wizard.AskConfirm("Does this service need a background worker?", false)
	if err != nil {
		return ctx, err
	}
	ctx.HasWorker = hasWorker

	// Q8: Worker port (conditional)
	if hasWorker {
		defaultWorkerPort := ctx.ApiPort + 1
		wpStr, err := wizard.AskInput(fmt.Sprintf("Worker port (default: %d):", defaultWorkerPort), strconv.Itoa(defaultWorkerPort), nil)
		if err != nil {
			return ctx, err
		}
		ctx.WorkerPort, _ = strconv.Atoi(strings.TrimSpace(wpStr))
	}

	// Q9: Domain entities
	entitiesStr, err := wizard.AskInput("Domain entities (PascalCase, comma-separated, e.g. \"User, Order, Payment\"):", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.Entities = wizard.ParseCommaSeparated(entitiesStr)
	ctx.EntityCount = len(ctx.Entities)

	// Q10: Core operations
	ops, err := wizard.AskInput("Core operations (describe main functionality):", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.CoreOperations = ops

	// Q11: Data stores
	dataStores, err := wizard.AskCheckbox("Data stores:", []wizard.Option{
		{Label: "PostgreSQL", Value: "postgresql"},
		{Label: "Redis", Value: "redis"},
		{Label: "RabbitMQ", Value: "rabbitmq"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.DataStores = dataStores
	for _, ds := range dataStores {
		switch ds {
		case "postgresql":
			ctx.HasPostgres = true
		case "redis":
			ctx.HasRedis = true
		case "rabbitmq":
			ctx.HasRabbitMQ = true
		}
	}

	// Q11b: RabbitMQ details (conditional)
	if ctx.HasRabbitMQ {
		exchange, err := wizard.AskInput("RabbitMQ exchange name:", ctx.ServiceName, nil)
		if err != nil {
			return ctx, err
		}
		ctx.ExchangeName = exchange

		defaultQueues := fmt.Sprintf("%s.deliveries, %s.retry, %s.dlq", exchange, exchange, exchange)
		queuesStr, err := wizard.AskInput("Queue names (comma-separated):", defaultQueues, nil)
		if err != nil {
			return ctx, err
		}
		ctx.Queues = wizard.ParseCommaSeparated(queuesStr)
	}

	// Q12: Auth requirements
	auth, err := wizard.AskCheckbox("Auth requirements:", []wizard.Option{
		{Label: "JWT (Bearer token)", Value: "jwt"},
		{Label: "X-Internal-API-Key", Value: "internal-key"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.AuthRequirements = auth
	for _, a := range auth {
		switch a {
		case "jwt":
			ctx.HasJWT = true
		case "internal-key":
			ctx.HasInternalKey = true
		}
	}

	// Q13: Special patterns
	patterns, err := wizard.AskCheckbox("Special patterns:", []wizard.Option{
		{Label: "HMAC signatures", Value: "hmac"},
		{Label: "Retry with exponential backoff", Value: "retry"},
		{Label: "Inbound webhooks", Value: "inbound-webhooks"},
		{Label: "Rate limiting", Value: "rate-limiting"},
		{Label: "Redis caching", Value: "caching"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.SpecialPatterns = patterns
	for _, p := range patterns {
		switch p {
		case "hmac":
			ctx.HasHMAC = true
		case "retry":
			ctx.HasRetry = true
		case "inbound-webhooks":
			ctx.HasInboundWebhooks = true
		case "rate-limiting":
			ctx.HasRateLimit = true
		case "caching":
			ctx.HasCaching = true
		}
	}

	// Q13b: Dead letter mechanism (conditional on retry)
	if ctx.HasRetry {
		if ctx.HasRabbitMQ {
			dlm, err := wizard.AskSelect("Dead-letter mechanism:", []wizard.Option{
				{Label: "RabbitMQ DLQ", Value: "rabbitmq-dlq"},
				{Label: "PostgreSQL status field", Value: "postgresql-status-field"},
			})
			if err != nil {
				return ctx, err
			}
			ctx.DeadLetterMechanism = dlm
		} else {
			ctx.DeadLetterMechanism = "postgresql-status-field"
		}
		ctx.RetryEntity = DeriveRetryEntity(ctx.Entities)
		ctx.RetryAttemptField = "attempt"
	}

	// Q14: External dependencies
	depsStr, err := wizard.AskInput("External originating project service dependencies (comma-separated, or empty):", "", nil)
	if err != nil {
		return ctx, err
	}
	if strings.TrimSpace(depsStr) != "" {
		ctx.ExternalDependencies = wizard.ParseCommaSeparated(depsStr)
	}

	// Q15: Module path
	defaultModule := fmt.Sprintf("github.com/originating-project/%s", ctx.ServiceName)
	if ctx.IsNode {
		defaultModule = fmt.Sprintf("@originating-project/%s", ctx.ServiceName)
	}
	modulePath, err := wizard.AskInput("Module path:", defaultModule, nil)
	if err != nil {
		return ctx, err
	}
	ctx.ModulePath = normalizeModulePath(modulePath)

	// Q16: Cycle config
	cycleConfig := CalculateCycles(ctx.Entities, ctx.SpecialPatterns)
	fmt.Printf("\n  Auto-calculated cycles: max=%d, dev=%d, test=%d, review=%d\n",
		cycleConfig.MaxCycles, cycleConfig.DeveloperMaxTurns, cycleConfig.TesterMaxTurns, cycleConfig.ReviewerMaxTurns)

	acceptCycles, err := wizard.AskConfirm("Accept these cycle settings?", true)
	if err != nil {
		return ctx, err
	}
	if acceptCycles {
		ctx.MaxCycles = cycleConfig.MaxCycles
		ctx.DeveloperMaxTurns = cycleConfig.DeveloperMaxTurns
		ctx.TesterMaxTurns = cycleConfig.TesterMaxTurns
		ctx.ReviewerMaxTurns = cycleConfig.ReviewerMaxTurns
	} else {
		// Manual override — propagate errors and guard against non-positive
		// values so loop.sh is never rendered with zeros.
		ctx.MaxCycles, err = askCycleValue("Max cycles:", cycleConfig.MaxCycles)
		if err != nil {
			return ctx, fmt.Errorf("reading max cycles: %w", err)
		}

		ctx.DeveloperMaxTurns, err = askCycleValue("Developer max turns:", cycleConfig.DeveloperMaxTurns)
		if err != nil {
			return ctx, fmt.Errorf("reading developer max turns: %w", err)
		}

		ctx.TesterMaxTurns, err = askCycleValue("Tester max turns:", cycleConfig.TesterMaxTurns)
		if err != nil {
			return ctx, fmt.Errorf("reading tester max turns: %w", err)
		}

		ctx.ReviewerMaxTurns, err = askCycleValue("Reviewer max turns:", cycleConfig.ReviewerMaxTurns)
		if err != nil {
			return ctx, fmt.Errorf("reading reviewer max turns: %w", err)
		}
	}

	return ctx, nil
}

// RunGeneralWizard runs the simplified general project wizard.
func RunGeneralWizard() (ProjectContext, error) {
	var ctx ProjectContext
	ctx.ProjectType = "general"
	ctx.IsInternal = false
	ctx.CreatedDate = time.Now().Format("2006-01-02")

	// Q1: Project name
	name, err := wizard.AskInput("Project name (kebab-case):", "", wizard.ValidateKebabCase)
	if err != nil {
		return ctx, err
	}
	ctx.ServiceName = name
	ctx.ServiceNamePascal = wizard.ToPascalCase(name)

	// Q2: Tech stack
	tech, err := wizard.AskSelect("Tech stack:", []wizard.Option{
		{Label: "Go", Value: "go"},
		{Label: "Node.js", Value: "node"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.Tech = tech
	ctx.IsGo = tech == "go"
	ctx.IsNode = tech == "node"
	if ctx.IsNode {
		ctx.NodeArchitecture = "4-layer"
	}

	// Q3 (Go only): Go version
	if ctx.IsGo {
		goVer, goErr := wizard.AskInput("Go version (for Dockerfile base image):", "1.24", wizard.ValidateMinLength(1))
		if goErr != nil {
			return ctx, goErr
		}
		ctx.GoVersion = strings.TrimSpace(goVer)
	}

	// Q4: Description
	desc, err := wizard.AskInput("Project description:", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.Description = desc

	// Q5: API port
	const defaultGeneralPort = 8080
	portStr, err := wizard.AskInput("API port:", strconv.Itoa(defaultGeneralPort), wizard.ValidatePort)
	if err != nil {
		return ctx, err
	}
	if p, convErr := strconv.Atoi(strings.TrimSpace(portStr)); convErr == nil {
		ctx.ApiPort = p
	} else {
		ctx.ApiPort = defaultGeneralPort
	}

	// Q5: Worker?
	hasWorker, err := wizard.AskConfirm("Does this project need a background worker?", false)
	if err != nil {
		return ctx, err
	}
	ctx.HasWorker = hasWorker
	if hasWorker {
		defaultWP := strconv.Itoa(ctx.ApiPort + 1)
		wpStr, wpErr := wizard.AskInput("Worker port:", defaultWP, wizard.ValidatePort)
		if wpErr != nil {
			return ctx, wpErr
		}
		if p, convErr := strconv.Atoi(strings.TrimSpace(wpStr)); convErr == nil {
			ctx.WorkerPort = p
		} else {
			ctx.WorkerPort = ctx.ApiPort + 1
		}
	}

	// Q6: Domain entities
	entitiesStr, err := wizard.AskInput("Domain entities (PascalCase, comma-separated):", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.Entities = wizard.ParseCommaSeparated(entitiesStr)
	ctx.EntityCount = len(ctx.Entities)

	// Q7: Core operations
	ops, err := wizard.AskInput("Core operations:", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.CoreOperations = ops

	// Q8: Data stores
	dataStores, err := wizard.AskCheckbox("Data stores:", []wizard.Option{
		{Label: "PostgreSQL", Value: "postgresql"},
		{Label: "Redis", Value: "redis"},
		{Label: "RabbitMQ", Value: "rabbitmq"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.DataStores = dataStores
	for _, ds := range dataStores {
		switch ds {
		case "postgresql":
			ctx.HasPostgres = true
		case "redis":
			ctx.HasRedis = true
		case "rabbitmq":
			ctx.HasRabbitMQ = true
		}
	}

	// Q9: Module path
	defaultModule := fmt.Sprintf("github.com/yourorg/%s", ctx.ServiceName)
	if ctx.IsNode {
		defaultModule = fmt.Sprintf("@yourorg/%s", ctx.ServiceName)
	}
	modulePath, err := wizard.AskInput("Module path:", defaultModule, nil)
	if err != nil {
		return ctx, err
	}
	ctx.ModulePath = normalizeModulePath(modulePath)

	// Auto-calculate cycles
	cycleConfig := CalculateCycles(ctx.Entities, nil)
	ctx.MaxCycles = cycleConfig.MaxCycles
	ctx.DeveloperMaxTurns = cycleConfig.DeveloperMaxTurns
	ctx.TesterMaxTurns = cycleConfig.TesterMaxTurns
	ctx.ReviewerMaxTurns = cycleConfig.ReviewerMaxTurns

	return ctx, nil
}

// RunGeneralWizardWithContext runs the general wizard with pre-filled identity/tech fields.
// Skips Q1 (project name) and Q2 (tech stack), asks everything else.
func RunGeneralWizardWithContext(ctx ProjectContext) (ProjectContext, error) {
	// Go version (if Go project, Tech was pre-filled by caller)
	if ctx.IsGo {
		goVer, goErr := wizard.AskInput("Go version (for Dockerfile base image):", "1.24", wizard.ValidateMinLength(1))
		if goErr != nil {
			return ctx, goErr
		}
		ctx.GoVersion = strings.TrimSpace(goVer)
	}

	// Q3: Description
	desc, err := wizard.AskInput("Project description:", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.Description = desc

	// Q4: API port
	const defaultGeneralPort = 8080
	portStr, err := wizard.AskInput("API port:", strconv.Itoa(defaultGeneralPort), wizard.ValidatePort)
	if err != nil {
		return ctx, err
	}
	if p, convErr := strconv.Atoi(strings.TrimSpace(portStr)); convErr == nil {
		ctx.ApiPort = p
	} else {
		ctx.ApiPort = defaultGeneralPort
	}

	// Q5: Worker?
	hasWorker, err := wizard.AskConfirm("Does this project need a background worker?", false)
	if err != nil {
		return ctx, err
	}
	ctx.HasWorker = hasWorker
	if hasWorker {
		defaultWP := strconv.Itoa(ctx.ApiPort + 1)
		wpStr, wpErr := wizard.AskInput("Worker port:", defaultWP, wizard.ValidatePort)
		if wpErr != nil {
			return ctx, wpErr
		}
		if p, convErr := strconv.Atoi(strings.TrimSpace(wpStr)); convErr == nil {
			ctx.WorkerPort = p
		} else {
			ctx.WorkerPort = ctx.ApiPort + 1
		}
	}

	// Q6: Domain entities
	entitiesStr, err := wizard.AskInput("Domain entities (PascalCase, comma-separated):", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.Entities = wizard.ParseCommaSeparated(entitiesStr)
	ctx.EntityCount = len(ctx.Entities)

	// Q7: Core operations
	ops, err := wizard.AskInput("Core operations:", "", nil)
	if err != nil {
		return ctx, err
	}
	ctx.CoreOperations = ops

	// Q8: Data stores
	dataStores, err := wizard.AskCheckbox("Data stores:", []wizard.Option{
		{Label: "PostgreSQL", Value: "postgresql"},
		{Label: "Redis", Value: "redis"},
		{Label: "RabbitMQ", Value: "rabbitmq"},
	})
	if err != nil {
		return ctx, err
	}
	ctx.DataStores = dataStores
	for _, ds := range dataStores {
		switch ds {
		case "postgresql":
			ctx.HasPostgres = true
		case "redis":
			ctx.HasRedis = true
		case "rabbitmq":
			ctx.HasRabbitMQ = true
		}
	}

	// Q9: Module path
	defaultModule := fmt.Sprintf("github.com/yourorg/%s", ctx.ServiceName)
	if ctx.IsNode {
		defaultModule = fmt.Sprintf("@yourorg/%s", ctx.ServiceName)
	}
	modulePath, err := wizard.AskInput("Module path:", defaultModule, nil)
	if err != nil {
		return ctx, err
	}
	ctx.ModulePath = normalizeModulePath(modulePath)

	// Auto-calculate cycles
	cycleConfig := CalculateCycles(ctx.Entities, nil)
	ctx.MaxCycles = cycleConfig.MaxCycles
	ctx.DeveloperMaxTurns = cycleConfig.DeveloperMaxTurns
	ctx.TesterMaxTurns = cycleConfig.TesterMaxTurns
	ctx.ReviewerMaxTurns = cycleConfig.ReviewerMaxTurns

	return ctx, nil
}

// normalizeModulePath strips common URL prefixes and trailing slashes from a module path.
// Users naturally type full GitHub URLs like "https://github.com/org/repo" — this
// normalizes to "github.com/org/repo".
func normalizeModulePath(path string) string {
	path = strings.TrimPrefix(path, "https://")
	path = strings.TrimPrefix(path, "http://")
	path = strings.TrimRight(path, "/")
	return path
}
