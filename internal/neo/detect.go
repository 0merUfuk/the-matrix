package neo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
)

// DetectionResult holds everything detected from scanning a project directory.
type DetectionResult struct {
	Language         string   // "go", "nodejs", "python", "flutter", "rust", "ruby"
	Framework        string   // "chi", "gin", "echo", "express", "fastapi", "django", etc.
	ArchStyle        string   // "layered", "feature-based", "flat", "hybrid"
	TestingFramework string   // "ginkgo", "jest", "pytest", "rspec", "go-test"
	DataLayer        string   // "bun", "gorm", "prisma", "sqlalchemy", "none"
	HasQueue         bool
	Observability    []string // ["zerolog", "sentry", "newrelic"]
	IsInternal         bool
	ProjectType      string // "single-app", "monorepo", "multi-repo-microservices"
}

// RunDetection executes all detection functions against a project directory
// and returns the combined result.
func RunDetection(projectPath string) DetectionResult {
	lang := detectLanguage(projectPath)
	return DetectionResult{
		Language:         lang,
		Framework:        detectFramework(projectPath, lang),
		ArchStyle:        detectArchStyle(projectPath),
		TestingFramework: detectTestingFramework(projectPath, lang),
		DataLayer:        detectDataLayer(projectPath, lang),
		HasQueue:         detectHasQueue(projectPath, lang),
		Observability:    detectObservability(projectPath, lang),
		IsInternal:         detectIsInternal(projectPath),
		ProjectType:      detectProjectType(projectPath),
	}
}

// detectLanguage checks for project manifest files at root level to determine
// the primary language.
func detectLanguage(projectPath string) string {
	checks := []struct {
		file string
		lang string
	}{
		{"go.mod", "go"},
		{"package.json", "nodejs"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
		{"pubspec.yaml", "flutter"},
		{"Cargo.toml", "rust"},
		{"Gemfile", "ruby"},
	}

	for _, c := range checks {
		if config.FileExists(filepath.Join(projectPath, c.file)) {
			return c.lang
		}
	}
	return "unknown"
}

// detectFramework parses the project's dependency manifest (go.mod or
// package.json) to identify the web framework in use.
func detectFramework(projectPath, language string) string {
	switch language {
	case "go":
		return detectGoFramework(projectPath)
	case "nodejs":
		return detectNodeFramework(projectPath)
	case "python":
		return detectPythonFramework(projectPath)
	case "ruby":
		return "rails" // reasonable default
	case "flutter":
		return "flutter"
	case "rust":
		return detectRustFramework(projectPath)
	}
	return "unknown"
}

func detectGoFramework(projectPath string) string {
	content := readFileQuiet(filepath.Join(projectPath, "go.mod"))
	if content == "" {
		return "unknown"
	}

	frameworks := []struct {
		module string
		name   string
	}{
		{"github.com/go-chi/chi", "chi"},
		{"github.com/gin-gonic/gin", "gin"},
		{"github.com/labstack/echo", "echo"},
		{"github.com/gofiber/fiber", "fiber"},
		{"github.com/gorilla/mux", "gorilla-mux"},
		{"net/http", "net-http"},
	}

	for _, fw := range frameworks {
		if strings.Contains(content, fw.module) {
			// Try to extract version for chi/gin/echo
			version := extractGoModVersion(content, fw.module)
			if version != "" {
				return fw.name + " " + version
			}
			return fw.name
		}
	}
	return "net-http"
}

func detectNodeFramework(projectPath string) string {
	deps := readPackageJSONDeps(filepath.Join(projectPath, "package.json"))
	if deps == nil {
		return "unknown"
	}

	frameworks := []struct {
		pkg  string
		name string
	}{
		{"next", "Next.js"},
		{"express", "express"},
		{"fastify", "fastify"},
		{"@nestjs/core", "NestJS"},
		{"koa", "koa"},
		{"hono", "hono"},
		{"nuxt", "Nuxt"},
	}

	for _, fw := range frameworks {
		if v, ok := deps[fw.pkg]; ok {
			return fw.name + " " + cleanVersion(v)
		}
	}
	return "unknown"
}

func detectPythonFramework(projectPath string) string {
	// Check requirements.txt
	content := readFileQuiet(filepath.Join(projectPath, "requirements.txt"))
	lowerContent := strings.ToLower(content)

	// Also check pyproject.toml
	pyproject := readFileQuiet(filepath.Join(projectPath, "pyproject.toml"))
	lowerPyproject := strings.ToLower(pyproject)

	combined := lowerContent + "\n" + lowerPyproject

	frameworks := []struct {
		marker string
		name   string
	}{
		{"fastapi", "FastAPI"},
		{"django", "Django"},
		{"flask", "Flask"},
		{"starlette", "Starlette"},
	}

	for _, fw := range frameworks {
		if strings.Contains(combined, fw.marker) {
			return fw.name
		}
	}
	return "unknown"
}

func detectRustFramework(projectPath string) string {
	content := readFileQuiet(filepath.Join(projectPath, "Cargo.toml"))
	lowerContent := strings.ToLower(content)

	frameworks := []struct {
		marker string
		name   string
	}{
		{"actix-web", "actix-web"},
		{"axum", "axum"},
		{"rocket", "rocket"},
		{"warp", "warp"},
	}

	for _, fw := range frameworks {
		if strings.Contains(lowerContent, fw.marker) {
			return fw.name
		}
	}
	return "unknown"
}

// detectArchStyle checks directory structure for common architectural patterns.
func detectArchStyle(projectPath string) string {
	hasCmdDir := config.IsDir(filepath.Join(projectPath, "cmd"))
	hasInternalDir := config.IsDir(filepath.Join(projectPath, "internal"))

	// Go standard layout: cmd/ + internal/
	if hasCmdDir && hasInternalDir {
		return "layered"
	}

	// MVC: src/controllers/ or controllers/
	if config.IsDir(filepath.Join(projectPath, "src", "controllers")) ||
		config.IsDir(filepath.Join(projectPath, "controllers")) {
		return "layered"
	}

	// Feature-based: src/features/ or src/modules/
	if config.IsDir(filepath.Join(projectPath, "src", "features")) ||
		config.IsDir(filepath.Join(projectPath, "src", "modules")) {
		return "feature-based"
	}

	// Layered with src/services/
	if config.IsDir(filepath.Join(projectPath, "src", "services")) {
		return "layered"
	}

	// Rails/Flutter conventions
	if config.IsDir(filepath.Join(projectPath, "app")) && config.IsDir(filepath.Join(projectPath, "lib")) {
		return "layered"
	}

	// Flat structure (no clear pattern)
	return "flat"
}

// detectTestingFramework determines the test framework from dependencies.
func detectTestingFramework(projectPath, language string) string {
	switch language {
	case "go":
		content := readFileQuiet(filepath.Join(projectPath, "go.mod"))
		if strings.Contains(content, "github.com/onsi/ginkgo") {
			return "ginkgo"
		}
		if strings.Contains(content, "github.com/stretchr/testify") {
			return "testify"
		}
		return "go-test"

	case "nodejs":
		deps := readPackageJSONAllDeps(filepath.Join(projectPath, "package.json"))
		if deps == nil {
			return "unknown"
		}
		if _, ok := deps["jest"]; ok {
			return "jest"
		}
		if _, ok := deps["vitest"]; ok {
			return "vitest"
		}
		if _, ok := deps["mocha"]; ok {
			return "mocha"
		}
		return "unknown"

	case "python":
		content := readFileQuiet(filepath.Join(projectPath, "requirements.txt"))
		pyproject := readFileQuiet(filepath.Join(projectPath, "pyproject.toml"))
		combined := strings.ToLower(content + "\n" + pyproject)
		if strings.Contains(combined, "pytest") {
			return "pytest"
		}
		return "unittest"

	case "ruby":
		content := readFileQuiet(filepath.Join(projectPath, "Gemfile"))
		if strings.Contains(strings.ToLower(content), "rspec") {
			return "rspec"
		}
		return "minitest"

	case "rust":
		return "cargo-test"

	case "flutter":
		return "flutter-test"
	}
	return "unknown"
}

// detectDataLayer determines the ORM / data layer from dependencies.
func detectDataLayer(projectPath, language string) string {
	switch language {
	case "go":
		content := readFileQuiet(filepath.Join(projectPath, "go.mod"))
		orms := []struct {
			module string
			name   string
		}{
			{"github.com/uptrace/bun", "bun"},
			{"gorm.io/gorm", "gorm"},
			{"github.com/jmoiron/sqlx", "sqlx"},
			{"entgo.io/ent", "ent"},
			{"github.com/jackc/pgx", "pgx"},
		}
		for _, orm := range orms {
			if strings.Contains(content, orm.module) {
				return orm.name
			}
		}

	case "nodejs":
		deps := readPackageJSONAllDeps(filepath.Join(projectPath, "package.json"))
		if deps == nil {
			return "none"
		}
		orms := []struct {
			pkg  string
			name string
		}{
			{"prisma", "prisma"},
			{"@prisma/client", "prisma"},
			{"typeorm", "typeorm"},
			{"sequelize", "sequelize"},
			{"knex", "knex"},
			{"drizzle-orm", "drizzle"},
			{"mongoose", "mongoose"},
		}
		for _, orm := range orms {
			if _, ok := deps[orm.pkg]; ok {
				return orm.name
			}
		}

	case "python":
		content := readFileQuiet(filepath.Join(projectPath, "requirements.txt"))
		pyproject := readFileQuiet(filepath.Join(projectPath, "pyproject.toml"))
		combined := strings.ToLower(content + "\n" + pyproject)
		if strings.Contains(combined, "sqlalchemy") {
			return "sqlalchemy"
		}
		if strings.Contains(combined, "django") {
			return "django-orm"
		}
		if strings.Contains(combined, "tortoise") {
			return "tortoise-orm"
		}

	case "ruby":
		return "activerecord"
	}

	return "none"
}

// detectHasQueue looks for message queue dependencies.
func detectHasQueue(projectPath, language string) bool {
	switch language {
	case "go":
		content := readFileQuiet(filepath.Join(projectPath, "go.mod"))
		return strings.Contains(content, "github.com/rabbitmq/amqp091-go") ||
			strings.Contains(content, "github.com/streadway/amqp") ||
			strings.Contains(content, "github.com/confluentinc/confluent-kafka-go") ||
			strings.Contains(content, "github.com/nats-io/nats.go")

	case "nodejs":
		deps := readPackageJSONAllDeps(filepath.Join(projectPath, "package.json"))
		if deps == nil {
			return false
		}
		queuePkgs := []string{"amqplib", "bull", "bullmq", "kafkajs", "nats"}
		for _, pkg := range queuePkgs {
			if _, ok := deps[pkg]; ok {
				return true
			}
		}

	case "python":
		content := readFileQuiet(filepath.Join(projectPath, "requirements.txt"))
		pyproject := readFileQuiet(filepath.Join(projectPath, "pyproject.toml"))
		combined := strings.ToLower(content + "\n" + pyproject)
		return strings.Contains(combined, "celery") ||
			strings.Contains(combined, "pika") ||
			strings.Contains(combined, "kombu")
	}

	return false
}

// detectObservability identifies observability tools from dependencies.
func detectObservability(projectPath, language string) []string {
	var obs []string

	switch language {
	case "go":
		content := readFileQuiet(filepath.Join(projectPath, "go.mod"))
		if strings.Contains(content, "github.com/rs/zerolog") {
			obs = append(obs, "zerolog")
		}
		if strings.Contains(content, "go.uber.org/zap") {
			obs = append(obs, "zap")
		}
		if strings.Contains(content, "github.com/getsentry/sentry-go") {
			obs = append(obs, "sentry")
		}
		if strings.Contains(content, "github.com/newrelic/go-agent") {
			obs = append(obs, "newrelic")
		}
		if strings.Contains(content, "go.opentelemetry.io") {
			obs = append(obs, "opentelemetry")
		}

	case "nodejs":
		deps := readPackageJSONAllDeps(filepath.Join(projectPath, "package.json"))
		if deps == nil {
			return obs
		}
		if _, ok := deps["pino"]; ok {
			obs = append(obs, "pino")
		}
		if _, ok := deps["winston"]; ok {
			obs = append(obs, "winston")
		}
		if _, ok := deps["@sentry/node"]; ok {
			obs = append(obs, "sentry")
		}
		if _, ok := deps["newrelic"]; ok {
			obs = append(obs, "newrelic")
		}

	case "python":
		content := readFileQuiet(filepath.Join(projectPath, "requirements.txt"))
		pyproject := readFileQuiet(filepath.Join(projectPath, "pyproject.toml"))
		combined := strings.ToLower(content + "\n" + pyproject)
		if strings.Contains(combined, "sentry-sdk") {
			obs = append(obs, "sentry")
		}
		if strings.Contains(combined, "structlog") {
			obs = append(obs, "structlog")
		}
	}

	return obs
}

// detectIsInternal checks if the project is a originating project project by looking for
// originating project markers in CLAUDE.md or docker-compose.yml.
func detectIsInternal(projectPath string) bool {
	// Check CLAUDE.md
	content := readFileQuiet(filepath.Join(projectPath, "CLAUDE.md"))
	if strings.Contains(content, "originating project") {
		return true
	}

	// Check docker-compose files for shared infra naming convention
	composeFiles := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, f := range composeFiles {
		content := readFileQuiet(filepath.Join(projectPath, f))
		if strings.Contains(strings.ToLower(content), "internal-") {
			return true
		}
	}

	return false
}

// detectProjectType determines whether the project is a single-app, monorepo,
// or multi-repo microservices setup by checking for multiple manifest files.
func detectProjectType(projectPath string) string {
	// Check for multiple go.mod or package.json files in subdirectories (1 level deep)
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return "single-app"
	}

	manifests := []string{"go.mod", "package.json", "Cargo.toml", "pubspec.yaml"}
	subprojectCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip hidden directories and common non-project dirs
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			continue
		}

		for _, m := range manifests {
			if config.FileExists(filepath.Join(projectPath, name, m)) {
				subprojectCount++
				break
			}
		}
	}

	if subprojectCount >= 2 {
		return "monorepo"
	}
	return "single-app"
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// readFileQuiet reads a file and returns its content, or empty string on error.
func readFileQuiet(path string) string {
	content, err := config.ReadFileString(path)
	if err != nil {
		return ""
	}
	return content
}

// packageJSON is a minimal representation for dependency extraction.
type packageJSON struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// readPackageJSON parses a package.json file.
func readPackageJSON(path string) *packageJSON {
	content := readFileQuiet(path)
	if content == "" {
		return nil
	}
	var pkg packageJSON
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil
	}
	return &pkg
}

// readPackageJSONDeps returns production dependencies from package.json.
func readPackageJSONDeps(path string) map[string]string {
	pkg := readPackageJSON(path)
	if pkg == nil {
		return nil
	}
	return pkg.Dependencies
}

// readPackageJSONAllDeps returns combined production + dev dependencies.
func readPackageJSONAllDeps(path string) map[string]string {
	pkg := readPackageJSON(path)
	if pkg == nil {
		return nil
	}
	all := make(map[string]string)
	for k, v := range pkg.Dependencies {
		all[k] = v
	}
	for k, v := range pkg.DevDependencies {
		all[k] = v
	}
	return all
}

// extractGoModVersion extracts the version of a module from go.mod content.
// For example, given "github.com/go-chi/chi/v5 v5.0.12", returns "v5".
func extractGoModVersion(content, module string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, module) {
			continue
		}
		// Extract major version from path (e.g., /v5) or version string
		parts := strings.Fields(line)
		for _, part := range parts {
			if strings.Contains(part, module) {
				// Check for /vN suffix
				if idx := strings.LastIndex(part, "/v"); idx != -1 {
					return part[idx+1:]
				}
			}
			if strings.HasPrefix(part, "v") && !strings.Contains(part, "/") {
				// This is a version like v1.2.3 — extract major
				dotIdx := strings.Index(part, ".")
				if dotIdx != -1 {
					return part[:dotIdx]
				}
				return part
			}
		}
	}
	return ""
}

// cleanVersion strips semver prefixes like ^ ~ >= from version strings.
func cleanVersion(v string) string {
	v = strings.TrimLeft(v, "^~>=<! ")
	return v
}
