// Package neo implements the neo meta-CLI: ecosystem provisioning and maintenance.
package neo

// ProjectProfile is the central data structure — produced by neo's wizard,
// consumed by all downstream tools (oracle, morp, trinity).
type ProjectProfile struct {
	// Identity
	ProjectName string `json:"projectName"` // kebab-case
	ProjectType string `json:"projectType"` // "monorepo" | "multi-repo-microservices" | "single-app"
	TeamSize    string `json:"teamSize"`    // "solo" | "small" | "mid"

	// Stacks (one or more)
	Stacks []StackProfile `json:"stacks"`

	// Services (for multi-repo-microservices only)
	Services []ServiceProfile `json:"services,omitempty"`

	// Context
	IsInternal  bool   `json:"isInternal"`
	CreatedDate string `json:"createdDate"` // YYYY-MM-DD
	OutputPath  string `json:"outputPath"`  // absolute path to project root
}

// StackProfile describes a tech stack used in the project.
type StackProfile struct {
	Name             string `json:"name"`      // kebab-case (e.g. "go-internal")
	Language         string `json:"language"`  // "go" | "nodejs" | "python" | "flutter" | ...
	Framework        string `json:"framework"` // e.g. "chi v5", "Next.js 15", "FastAPI"
	ArchStyle        string `json:"archStyle"` // "layered" | "feature-based" | "flat" | "hybrid"
	TestingFramework string `json:"testingFramework"`
	HasQueue         bool   `json:"hasQueue"`
	DataLayer        string `json:"dataLayer"` // ORM name or "none"
}

// ServiceProfile describes a service in a multi-repo project.
type ServiceProfile struct {
	Name  string `json:"name"`  // service name
	Stack string `json:"stack"` // references StackProfile.Name
	Port  int    `json:"port"`  // API port
}
