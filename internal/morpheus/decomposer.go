package morpheus

import (
	"fmt"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/subprocess"
)

// DecomposeTasksWithClaude generates tasks/todo.md content using Claude subprocess.
func DecomposeTasksWithClaude(ctx *ProjectContext) (string, error) {
	contextDoc := BuildProjectContext(ctx)
	prompt := BuildTaskDecompositionPrompt(ctx)
	fullPrompt := fmt.Sprintf("CONTEXT:\n%s\n\n%s", contextDoc, prompt)

	result, err := subprocess.CallClaude(fullPrompt, subprocess.Options{
		Model:     "claude-sonnet-4-6",
		MaxTurns:  10,
		TimeoutMs: 5 * 60 * 1000,
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// BuildProjectContext builds a KEY: VALUE string describing the project for Claude.
func BuildProjectContext(ctx *ProjectContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("SERVICE: %s\n", ctx.ServiceName))
	sb.WriteString(fmt.Sprintf("DESCRIPTION: %s\n", ctx.Description))
	sb.WriteString(fmt.Sprintf("TECH: %s\n", strings.ToUpper(ctx.Tech)))
	sb.WriteString(fmt.Sprintf("MODULE_PATH: %s\n", ctx.ModulePath))

	if ctx.IsInternal {
		sb.WriteString("PROJECT_TYPE: Internal Microservice\n")
	} else {
		sb.WriteString("PROJECT_TYPE: General Project\n")
	}

	sb.WriteString(fmt.Sprintf("\n## Ports\nAPI_PORT: %d\n", ctx.ApiPort))
	if ctx.HasWorker {
		sb.WriteString(fmt.Sprintf("WORKER_PORT: %d\n", ctx.WorkerPort))
	} else {
		sb.WriteString("WORKER: none\n")
	}

	sb.WriteString(fmt.Sprintf("\n## Domain\nENTITIES: %s\n", strings.Join(ctx.Entities, ", ")))
	sb.WriteString(fmt.Sprintf("CORE_OPERATIONS: %s\n", ctx.CoreOperations))
	if ctx.IsMigration {
		sb.WriteString(fmt.Sprintf("MIGRATION_SOURCE: %s\n", ctx.MigrationSource))
	} else {
		sb.WriteString("MIGRATION: fresh start\n")
	}

	sb.WriteString(fmt.Sprintf("\n## Data Stores\nDATA_STORES: %s\n", strings.Join(ctx.DataStores, ", ")))
	if ctx.HasRabbitMQ {
		sb.WriteString(fmt.Sprintf("EXCHANGE: %s\n", ctx.ExchangeName))
		sb.WriteString(fmt.Sprintf("QUEUES: %s\n", strings.Join(ctx.Queues, ", ")))
	}

	sb.WriteString("\n## Auth\n")
	if len(ctx.AuthRequirements) > 0 {
		sb.WriteString(fmt.Sprintf("AUTH: %s\n", strings.Join(ctx.AuthRequirements, ", ")))
	} else {
		sb.WriteString("AUTH: none\n")
	}

	sb.WriteString("\n## Special Patterns\n")
	if ctx.HasRetry {
		sb.WriteString("RETRY: exponential-backoff\n")
		sb.WriteString(fmt.Sprintf("RETRY_DEAD_LETTER: %s\n", ctx.DeadLetterMechanism))
		sb.WriteString(fmt.Sprintf("RETRY_ENTITY: %s\n", ctx.RetryEntity))
	}
	if ctx.HasHMAC {
		sb.WriteString("HMAC: sha256-request-signing\n")
	}
	if ctx.HasInboundWebhooks {
		sb.WriteString("INBOUND_WEBHOOKS: signature-verification-required\n")
	}
	if ctx.HasRateLimit {
		sb.WriteString("RATE_LIMIT: redis-sliding-window\n")
	}
	if ctx.HasCaching {
		sb.WriteString("CACHING: redis-cache-aside\n")
	}
	if len(ctx.SpecialPatterns) == 0 {
		sb.WriteString("PATTERNS: none\n")
	}

	if len(ctx.ExternalDependencies) > 0 {
		sb.WriteString(fmt.Sprintf("\n## External Dependencies\nDEPENDENCIES: %s\n", strings.Join(ctx.ExternalDependencies, ", ")))
	}

	sb.WriteString("\n## Architecture\n")
	if ctx.IsGo {
		sb.WriteString("4-layer architecture: Handler → Orchestration → Service → Repository\n")
		sb.WriteString("Testing: Ginkgo/Gomega + gomock\n")
		sb.WriteString("DI: Google Wire\n")
		sb.WriteString("ORM: Bun (uptrace/bun)\n")
	} else {
		sb.WriteString("4-layer architecture: Handler → Orchestration → Service → Repository\n")
		sb.WriteString("Testing: Jest + supertest\n")
		sb.WriteString("DI: Constructor injection\n")
		sb.WriteString("ORM: Prisma\n")
	}

	return sb.String()
}

// BuildTaskDecompositionPrompt creates the Claude prompt for generating tasks/todo.md.
func BuildTaskDecompositionPrompt(ctx *ProjectContext) string {
	techLabel := "Go"
	if ctx.IsNode {
		techLabel = "Node.js"
	}

	testFramework := "Ginkgo v2 Describe/Context/It + Gomega + gomock"
	if ctx.IsNode {
		testFramework = "Jest describe/it + supertest"
	}

	return fmt.Sprintf(`You are a senior %s engineer. Generate a comprehensive tasks/todo.md for the %s microservice.

Requirements:
- 50-80 specific, actionable tasks
- Checkbox format: "- [ ] Description"
- Phase headers: "## Phase X: ..."
- Dependency order: foundation → domain → service → handler → tests
- Include ALL: directory structure, interfaces, entities, repositories, services, handlers, config, tests
- Entity-specific tasks for: %s
- Tech-specific testing: %s
- Repository tests MANDATORY for all layers
- Granular: 1-3 turns per task
- No artifact counts — list actual names
- Environment success criteria: expected tables, indexes, rollback verification

Return ONLY the markdown content for tasks/todo.md.
Start with "# Tasks: %s"`, techLabel, ctx.ServiceName, strings.Join(ctx.Entities, ", "), testFramework, ctx.ServiceNamePascal)
}

// BuildSkeletonTodo generates a fallback skeleton tasks/todo.md when Claude is unavailable.
func BuildSkeletonTodo(ctx *ProjectContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Tasks: %s\n\n", ctx.ServiceNamePascal))
	sb.WriteString("> **Note**: Generated from skeleton template. Claude was unavailable. Consider re-running morpheus init or manually expanding these tasks.\n\n")

	sb.WriteString("## Phase 1: Foundation\n\n")
	if ctx.IsGo {
		sb.WriteString(fmt.Sprintf("- [ ] Initialize go.mod: `go mod init %s`\n", ctx.ModulePath))
	}
	sb.WriteString("- [ ] Create directory structure\n")
	sb.WriteString("- [ ] Set up dependencies and tooling\n")
	sb.WriteString("- [ ] Configure Makefile and .gitignore\n\n")

	sb.WriteString("## Phase 2: Infrastructure\n\n")
	if ctx.HasPostgres {
		sb.WriteString("- [ ] Set up PostgreSQL connection and migrations\n")
	}
	if ctx.HasRedis {
		sb.WriteString("- [ ] Set up Redis client and connection\n")
	}
	if ctx.HasRabbitMQ {
		sb.WriteString(fmt.Sprintf("- [ ] Set up RabbitMQ connection (exchange: %s)\n", ctx.ExchangeName))
		for _, q := range ctx.Queues {
			sb.WriteString(fmt.Sprintf("- [ ] Configure queue: %s\n", q))
		}
	}
	sb.WriteString("- [ ] Set up config loading (YAML + env vars)\n\n")

	sb.WriteString("## Phase 3: Domain\n\n")
	for _, entity := range ctx.Entities {
		sb.WriteString(fmt.Sprintf("- [ ] Define %s entity interface\n", entity))
		sb.WriteString(fmt.Sprintf("- [ ] Implement %s entity struct\n", entity))
	}
	sb.WriteString("\n")

	sb.WriteString("## Phase 4: Implementation\n\n")
	for _, entity := range ctx.Entities {
		sb.WriteString(fmt.Sprintf("- [ ] Implement %s repository\n", entity))
		sb.WriteString(fmt.Sprintf("- [ ] Implement %s service\n", entity))
		sb.WriteString(fmt.Sprintf("- [ ] Implement %s handler (CRUD endpoints)\n", entity))
		sb.WriteString(fmt.Sprintf("- [ ] Write %s repository tests\n", entity))
		sb.WriteString(fmt.Sprintf("- [ ] Write %s service tests\n", entity))
		sb.WriteString(fmt.Sprintf("- [ ] Write %s handler tests\n", entity))
	}
	sb.WriteString("\n")

	sb.WriteString("## Phase 5: Wire & Entry Points\n\n")
	sb.WriteString("- [ ] Set up cmd/api/main.go\n")
	if ctx.HasWorker {
		sb.WriteString("- [ ] Set up cmd/worker/main.go\n")
	}
	if ctx.IsGo {
		sb.WriteString("- [ ] Configure Wire dependency injection\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Phase 6: Finalization\n\n")
	sb.WriteString("- [ ] Health check endpoints\n")
	sb.WriteString("- [ ] Final integration tests\n")

	return sb.String()
}
