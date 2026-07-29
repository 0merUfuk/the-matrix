// Package oracle implements the oracle CLI commands: research, status, and inject.
package oracle

// StackContext is the central data structure — produced by the wizard or loaded from --config JSON.
type StackContext struct {
	StackName         string            `json:"stackName"`
	FrameworkVersion  string            `json:"frameworkVersion"`
	AppType           string            `json:"appType"`       // full-stack|api-only|frontend|cli|library
	ArchStyle         string            `json:"archStyle"`     // layered|feature-based|flat|hybrid
	StateApproach     string            `json:"stateApproach"` // server-side|client-state|hybrid|n/a
	DataLayer         string            `json:"dataLayer"`
	TestingFramework  string            `json:"testingFramework"`
	HasQueue          bool              `json:"hasQueue"`
	HasBackgroundJobs bool              `json:"hasBackgroundJobs"`
	SelectedTopics    []string          `json:"selectedTopics"`
	TopicCount        int               `json:"topicCount"`
	OutputPath        string            `json:"outputPath"`
	CreatedDate       string            `json:"createdDate"`
	GoldStandards     map[string]string `json:"-"` // Injected at render time, not from wizard/JSON
	GoldStandardsDir  string            `json:"-"` // Resolved from --gold-standards flag or convention dir
}

// Topic represents a single knowledge topic with display name.
type Topic struct {
	Value string
	Name  string
}

// AllTopics is the standard 17 research topics for a comprehensive knowledge base.
var AllTopics = []Topic{
	{Value: "00-core-principles", Name: "00 Core Principles"},
	{Value: "01-project-structure", Name: "01 Project Structure"},
	{Value: "02-layered-architecture", Name: "02 Layered Architecture"},
	{Value: "03-dependency-injection", Name: "03 Dependency Injection"},
	{Value: "04-error-handling", Name: "04 Error Handling"},
	{Value: "05-testing", Name: "05 Testing"},
	{Value: "06-database", Name: "06 Database"},
	{Value: "07-configuration", Name: "07 Configuration"},
	{Value: "08-api-design", Name: "08 API Design"},
	{Value: "09-security", Name: "09 Security"},
	{Value: "10-deployment", Name: "10 Deployment"},
	{Value: "11-redis-caching", Name: "11 Redis & Caching"},
	{Value: "12-queue-integration", Name: "12 Queue Integration"},
	{Value: "13-observability", Name: "13 Observability"},
	{Value: "15-api-documentation", Name: "15 API Documentation"},
	{Value: "16-documentation-standards", Name: "16 Documentation Standards"},
	{Value: "99-completion-protocol", Name: "99 Session Completion Protocol"},
}

// AllTopicValues returns just the value strings from AllTopics.
func AllTopicValues() []string {
	vals := make([]string, len(AllTopics))
	for i, t := range AllTopics {
		vals[i] = t.Value
	}
	return vals
}

// CycleGroup represents a group of topics assigned to a research cycle.
type CycleGroup struct {
	Num    int
	Label  string
	Topics []string
}

// CycleGroups defines the 4 fixed research cycles.
var CycleGroups = []CycleGroup{
	{Num: 1, Label: "Foundations (00–03)", Topics: []string{"00-core-principles", "01-project-structure", "02-layered-architecture", "03-dependency-injection"}},
	{Num: 2, Label: "Implementation (04–07)", Topics: []string{"04-error-handling", "05-testing", "06-database", "07-configuration"}},
	{Num: 3, Label: "API & Ops (08–11)", Topics: []string{"08-api-design", "09-security", "10-deployment", "11-redis-caching"}},
	{Num: 4, Label: "Advanced (12–16, 99)", Topics: []string{"12-queue-integration", "13-observability", "15-api-documentation", "16-documentation-standards", "99-completion-protocol"}},
}

// StatusMap maps status file values to human-readable display strings.
var StatusMap = map[string]struct {
	Text  string
	Color string // "dim", "yellow", "cyan", "magenta", "green", "red"
}{
	"INITIALIZED":        {Text: "Initialized — loop not started", Color: "dim"},
	"RESEARCH_CYCLE_1":   {Text: "Research — Cycle 1 (Foundations)", Color: "yellow"},
	"RESEARCH_CYCLE_2":   {Text: "Research — Cycle 2 (Implementation)", Color: "yellow"},
	"RESEARCH_CYCLE_3":   {Text: "Research — Cycle 3 (API & Ops)", Color: "yellow"},
	"RESEARCH_CYCLE_4":   {Text: "Research — Cycle 4 (Advanced)", Color: "yellow"},
	"RESEARCH_COMPLETE":  {Text: "Research complete — awaiting human review gate", Color: "cyan"},
	"SYNTHESIS_RUNNING":  {Text: "Synthesis running...", Color: "magenta"},
	"SYNTHESIS_COMPLETE": {Text: "SYNTHESIS COMPLETE — output/ ready", Color: "green"},
	"STOPPED_BY_USER":    {Text: "Stopped by user", Color: "red"},
}
