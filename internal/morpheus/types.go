// Package morpheus implements the morpheus CLI commands: init, finalize, doctor, status, and report.
package morpheus

// ProjectContext is the central data structure — produced by wizards, consumed by all templates and generators.
// This is a flat structure matching the Node.js implementation.
type ProjectContext struct {
	// Identity
	ServiceName       string   `json:"serviceName"`
	ServiceNamePascal string   `json:"serviceNamePascal"`
	Description       string   `json:"description"`
	ModulePath        string   `json:"modulePath"`
	ProjectType       string   `json:"projectType"` // "internal" | "general"
	Tech              string   `json:"tech"`         // "go" | "node"
	IsGo              bool     `json:"isGo"`
	IsNode            bool     `json:"isNode"`
	IsInternal          bool     `json:"isInternal"`
	GoVersion         string   `json:"goVersion"` // Go version for Dockerfile base image, e.g. "1.24"
	CreatedDate       string   `json:"createdDate"`

	// Ports
	ApiPort    int  `json:"apiPort"`
	WorkerPort int  `json:"workerPort"` // 0 if no worker
	HasWorker  bool `json:"hasWorker"`

	// Domain
	Entities       []string `json:"entities"`
	EntityCount    int      `json:"entityCount"`
	CoreOperations string   `json:"coreOperations"`

	// Migration (originating project only)
	IsMigration     bool   `json:"isMigration"`
	MigrationSource string `json:"migrationSource"`

	// Data Stores
	DataStores   []string `json:"dataStores"`
	HasPostgres  bool     `json:"hasPostgres"`
	HasRedis     bool     `json:"hasRedis"`
	HasRabbitMQ  bool     `json:"hasRabbitMQ"`
	ExchangeName string   `json:"exchangeName"`
	Queues       []string `json:"queues"`

	// Auth (originating project only)
	HasJWT           bool     `json:"hasJWT"`
	HasInternalKey   bool     `json:"hasInternalKey"`
	AuthRequirements []string `json:"authRequirements"`

	// Special Patterns (originating project only)
	HasHMAC             bool     `json:"hasHMAC"`
	HasRetry            bool     `json:"hasRetry"`
	HasInboundWebhooks  bool     `json:"hasInboundWebhooks"`
	HasRateLimit        bool     `json:"hasRateLimit"`
	HasCaching          bool     `json:"hasCaching"`
	DeadLetterMechanism string   `json:"deadLetterMechanism"`
	RetryEntity         string   `json:"retryEntity"`
	RetryAttemptField   string   `json:"retryAttemptField"`
	SpecialPatterns     []string `json:"specialPatterns"`

	// External Dependencies (originating project only)
	ExternalDependencies []string `json:"externalDependencies"`

	// Cycle Config
	MaxCycles         int `json:"maxCycles"`
	DeveloperMaxTurns int `json:"developerMaxTurns"`
	TesterMaxTurns    int `json:"testerMaxTurns"`
	ReviewerMaxTurns  int `json:"reviewerMaxTurns"`

	// Node.js specific
	NodeArchitecture string `json:"nodeArchitecture"` // "4-layer" if Node.js
}

// CycleConfig holds auto-calculated or user-overridden cycle configuration.
type CycleConfig struct {
	MaxCycles         int
	DeveloperMaxTurns int
	TesterMaxTurns    int
	ReviewerMaxTurns  int
}

// CalculateCycles computes cycle config based on entity count and pattern complexity.
func CalculateCycles(entities []string, specialPatterns []string) CycleConfig {
	complexity := len(entities)
	for _, p := range specialPatterns {
		switch p {
		case "retry", "hmac", "inbound-webhooks":
			complexity += 2
		}
	}

	if complexity <= 6 {
		return CycleConfig{MaxCycles: 12, DeveloperMaxTurns: 28, TesterMaxTurns: 20, ReviewerMaxTurns: 12}
	}
	if complexity <= 12 {
		return CycleConfig{MaxCycles: 20, DeveloperMaxTurns: 35, TesterMaxTurns: 25, ReviewerMaxTurns: 15}
	}
	return CycleConfig{MaxCycles: 28, DeveloperMaxTurns: 40, TesterMaxTurns: 30, ReviewerMaxTurns: 18}
}

// DeriveRetryEntity picks the best entity for retry tracking from the entities list.
// Preference order: Execution > Delivery > Message > Event > Task > Job > first entity.
func DeriveRetryEntity(entities []string) string {
	preferred := []string{"Execution", "Delivery", "Message", "Event", "Task", "Job"}
	for _, p := range preferred {
		for _, e := range entities {
			if e == p {
				return e
			}
		}
	}
	if len(entities) > 0 {
		return entities[0]
	}
	return "Entity"
}
