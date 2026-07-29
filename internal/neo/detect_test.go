package neo

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// makeTempProject creates a temporary directory with the given files and
// returns the directory path. Files are written relative to the directory.
func makeTempProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// makeTempDirs creates empty subdirectories inside a temp dir.
func makeTempDirs(t *testing.T, base string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(base, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
}

// ── detectLanguage ────────────────────────────────────────────────────────────

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantLang string
	}{
		{
			name:     "Go project with go.mod",
			files:    map[string]string{"go.mod": "module example.com/foo\n\ngo 1.21\n"},
			wantLang: "go",
		},
		{
			name:     "Node.js project with package.json",
			files:    map[string]string{"package.json": `{"name":"foo","version":"1.0.0"}`},
			wantLang: "nodejs",
		},
		{
			name:     "Python project with requirements.txt",
			files:    map[string]string{"requirements.txt": "fastapi>=0.100\n"},
			wantLang: "python",
		},
		{
			name:     "Python project with pyproject.toml",
			files:    map[string]string{"pyproject.toml": "[tool.poetry]\nname = \"myapp\"\n"},
			wantLang: "python",
		},
		{
			name:     "Flutter project with pubspec.yaml",
			files:    map[string]string{"pubspec.yaml": "name: my_app\nsdkversion: '>=3.0.0'\n"},
			wantLang: "flutter",
		},
		{
			name:     "Rust project with Cargo.toml",
			files:    map[string]string{"Cargo.toml": "[package]\nname = \"foo\"\nversion = \"0.1.0\"\n"},
			wantLang: "rust",
		},
		{
			name:     "Ruby project with Gemfile",
			files:    map[string]string{"Gemfile": "source \"https://rubygems.org\"\ngem \"rails\"\n"},
			wantLang: "ruby",
		},
		{
			name:     "empty project returns unknown",
			files:    map[string]string{},
			wantLang: "unknown",
		},
		{
			name: "go.mod wins over package.json (priority order)",
			files: map[string]string{
				"go.mod":       "module example.com/foo\n",
				"package.json": `{"name":"foo"}`,
			},
			wantLang: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := detectLanguage(dir)
			if got != tt.wantLang {
				t.Errorf("detectLanguage() = %q, want %q", got, tt.wantLang)
			}
		})
	}
}

// ── detectFramework ───────────────────────────────────────────────────────────

func TestDetectGoFramework(t *testing.T) {
	tests := []struct {
		name     string
		gomod    string
		wantFw   string // prefix match is used for versioned names
		contains bool   // if true, check Contains instead of ==
	}{
		{
			name:     "chi framework",
			gomod:    "module foo\n\nrequire github.com/go-chi/chi/v5 v5.0.12\n",
			wantFw:   "chi",
			contains: true,
		},
		{
			name:     "gin framework",
			gomod:    "module foo\n\nrequire github.com/gin-gonic/gin v1.9.1\n",
			wantFw:   "gin",
			contains: true,
		},
		{
			name:     "echo framework",
			gomod:    "module foo\n\nrequire github.com/labstack/echo/v4 v4.12.0\n",
			wantFw:   "echo",
			contains: true,
		},
		{
			name:     "fiber framework",
			gomod:    "module foo\n\nrequire github.com/gofiber/fiber/v2 v2.52.0\n",
			wantFw:   "fiber",
			contains: true,
		},
		{
			name:     "gorilla-mux framework (slash sanitized to hyphen)",
			gomod:    "module foo\n\nrequire github.com/gorilla/mux v1.8.1\n",
			wantFw:   "gorilla-mux",
			contains: true,
		},
		{
			name:     "no known framework falls back to net-http",
			gomod:    "module foo\n\nrequire github.com/rs/zerolog v1.31.0\n",
			wantFw:   "net-http",
			contains: false,
		},
		{
			name:     "empty go.mod returns unknown",
			gomod:    "",
			wantFw:   "unknown",
			contains: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.gomod == "" {
				dir = t.TempDir() // no go.mod file
			} else {
				dir = makeTempProject(t, map[string]string{"go.mod": tt.gomod})
			}
			got := detectGoFramework(dir)
			if tt.contains {
				if len(got) < len(tt.wantFw) || got[:len(tt.wantFw)] != tt.wantFw {
					t.Errorf("detectGoFramework() = %q, want prefix %q", got, tt.wantFw)
				}
			} else {
				if got != tt.wantFw {
					t.Errorf("detectGoFramework() = %q, want %q", got, tt.wantFw)
				}
			}
		})
	}
}

func TestDetectNodeFramework(t *testing.T) {
	tests := []struct {
		name    string
		pkgJSON string
		wantFw  string
	}{
		{
			name:    "express framework",
			pkgJSON: `{"dependencies":{"express":"^4.18.0"}}`,
			wantFw:  "express",
		},
		{
			name:    "Next.js framework",
			pkgJSON: `{"dependencies":{"next":"14.0.0"}}`,
			wantFw:  "Next.js",
		},
		{
			name:    "NestJS framework",
			pkgJSON: `{"dependencies":{"@nestjs/core":"^10.0.0"}}`,
			wantFw:  "NestJS",
		},
		{
			name:    "fastify framework",
			pkgJSON: `{"dependencies":{"fastify":"^4.0.0"}}`,
			wantFw:  "fastify",
		},
		{
			name:    "no known framework returns unknown",
			pkgJSON: `{"dependencies":{"lodash":"^4.0.0"}}`,
			wantFw:  "unknown",
		},
		{
			name:    "missing package.json returns unknown",
			pkgJSON: "",
			wantFw:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.pkgJSON == "" {
				dir = t.TempDir()
			} else {
				dir = makeTempProject(t, map[string]string{"package.json": tt.pkgJSON})
			}
			got := detectNodeFramework(dir)
			// Contains check for versioned framework names
			if len(got) < len(tt.wantFw) || got[:len(tt.wantFw)] != tt.wantFw {
				t.Errorf("detectNodeFramework() = %q, want prefix %q", got, tt.wantFw)
			}
		})
	}
}

func TestDetectFramework_LanguageDispatch(t *testing.T) {
	tests := []struct {
		name     string
		language string
		files    map[string]string
		wantFw   string
	}{
		{
			name:     "ruby always returns rails",
			language: "ruby",
			files:    map[string]string{},
			wantFw:   "rails",
		},
		{
			name:     "flutter always returns flutter",
			language: "flutter",
			files:    map[string]string{},
			wantFw:   "flutter",
		},
		{
			name:     "rust with axum",
			language: "rust",
			files:    map[string]string{"Cargo.toml": "[dependencies]\naxum = \"0.7\"\n"},
			wantFw:   "axum",
		},
		{
			name:     "python with fastapi",
			language: "python",
			files:    map[string]string{"requirements.txt": "fastapi>=0.100\nuvicorn\n"},
			wantFw:   "FastAPI",
		},
		{
			name:     "unknown language returns unknown",
			language: "unknown",
			files:    map[string]string{},
			wantFw:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := detectFramework(dir, tt.language)
			if got != tt.wantFw {
				t.Errorf("detectFramework(%q) = %q, want %q", tt.language, got, tt.wantFw)
			}
		})
	}
}

func TestDetectGoFramework_FallbackHasNoSlash(t *testing.T) {
	// The fallback framework name must be path-safe (no "/" characters)
	// because it flows into stackName which is used as a directory/file path
	// component in generator.go (e.g., .claude/rules/{stackName}-services.md).
	dir := makeTempProject(t, map[string]string{
		"go.mod": "module foo\n\ngo 1.21\n\nrequire github.com/rs/zerolog v1.31.0\n",
	})
	fw := detectGoFramework(dir)
	if strings.Contains(fw, "/") {
		t.Errorf("detectGoFramework() fallback = %q, must not contain '/' (breaks path construction)", fw)
	}
}

// ── detectArchStyle ───────────────────────────────────────────────────────────

func TestDetectArchStyle(t *testing.T) {
	tests := []struct {
		name     string
		dirs     []string
		wantArch string
	}{
		{
			name:     "Go standard layout: cmd/ + internal/ => layered",
			dirs:     []string{"cmd", "internal"},
			wantArch: "layered",
		},
		{
			name:     "MVC src/controllers => layered",
			dirs:     []string{"src/controllers"},
			wantArch: "layered",
		},
		{
			name:     "MVC controllers at root => layered",
			dirs:     []string{"controllers"},
			wantArch: "layered",
		},
		{
			name:     "feature-based src/features => feature-based",
			dirs:     []string{"src/features"},
			wantArch: "feature-based",
		},
		{
			name:     "feature-based src/modules => feature-based",
			dirs:     []string{"src/modules"},
			wantArch: "feature-based",
		},
		{
			name:     "layered src/services => layered",
			dirs:     []string{"src/services"},
			wantArch: "layered",
		},
		{
			name:     "Rails/Flutter app + lib => layered",
			dirs:     []string{"app", "lib"},
			wantArch: "layered",
		},
		{
			name:     "flat: no recognizable structure",
			dirs:     []string{},
			wantArch: "flat",
		},
		{
			name:     "only cmd without internal is flat",
			dirs:     []string{"cmd"},
			wantArch: "flat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			makeTempDirs(t, dir, tt.dirs...)
			got := detectArchStyle(dir)
			if got != tt.wantArch {
				t.Errorf("detectArchStyle() = %q, want %q", got, tt.wantArch)
			}
		})
	}
}

// ── detectTestingFramework ────────────────────────────────────────────────────

func TestDetectTestingFramework(t *testing.T) {
	tests := []struct {
		name     string
		language string
		files    map[string]string
		wantFw   string
	}{
		{
			name:     "Go with ginkgo",
			language: "go",
			files:    map[string]string{"go.mod": "module foo\n\nrequire github.com/onsi/ginkgo/v2 v2.15.0\n"},
			wantFw:   "ginkgo",
		},
		{
			name:     "Go with testify",
			language: "go",
			files:    map[string]string{"go.mod": "module foo\n\nrequire github.com/stretchr/testify v1.9.0\n"},
			wantFw:   "testify",
		},
		{
			name:     "Go with no testing framework defaults to go-test",
			language: "go",
			files:    map[string]string{"go.mod": "module foo\n\ngo 1.21\n"},
			wantFw:   "go-test",
		},
		{
			name:     "Node.js with jest in devDependencies",
			language: "nodejs",
			files:    map[string]string{"package.json": `{"devDependencies":{"jest":"^29.0.0"}}`},
			wantFw:   "jest",
		},
		{
			name:     "Node.js with vitest",
			language: "nodejs",
			files:    map[string]string{"package.json": `{"devDependencies":{"vitest":"^1.0.0"}}`},
			wantFw:   "vitest",
		},
		{
			name:     "Node.js with mocha",
			language: "nodejs",
			files:    map[string]string{"package.json": `{"devDependencies":{"mocha":"^10.0.0"}}`},
			wantFw:   "mocha",
		},
		{
			name:     "Node.js with no recognized framework returns unknown",
			language: "nodejs",
			files:    map[string]string{"package.json": `{"devDependencies":{"eslint":"^8.0.0"}}`},
			wantFw:   "unknown",
		},
		{
			name:     "Python with pytest in requirements.txt",
			language: "python",
			files:    map[string]string{"requirements.txt": "pytest>=7.0\nfastapi\n"},
			wantFw:   "pytest",
		},
		{
			name:     "Python with no pytest defaults to unittest",
			language: "python",
			files:    map[string]string{"requirements.txt": "fastapi\n"},
			wantFw:   "unittest",
		},
		{
			name:     "Ruby with rspec in Gemfile",
			language: "ruby",
			files:    map[string]string{"Gemfile": "gem \"rspec\"\ngem \"rails\"\n"},
			wantFw:   "rspec",
		},
		{
			name:     "Ruby with no rspec defaults to minitest",
			language: "ruby",
			files:    map[string]string{"Gemfile": "gem \"rails\"\n"},
			wantFw:   "minitest",
		},
		{
			name:     "Rust always returns cargo-test",
			language: "rust",
			files:    map[string]string{},
			wantFw:   "cargo-test",
		},
		{
			name:     "Flutter always returns flutter-test",
			language: "flutter",
			files:    map[string]string{},
			wantFw:   "flutter-test",
		},
		{
			name:     "unknown language returns unknown",
			language: "unknown",
			files:    map[string]string{},
			wantFw:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := detectTestingFramework(dir, tt.language)
			if got != tt.wantFw {
				t.Errorf("detectTestingFramework(%q) = %q, want %q", tt.language, got, tt.wantFw)
			}
		})
	}
}

// ── detectDataLayer ───────────────────────────────────────────────────────────

func TestDetectDataLayer(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		files     map[string]string
		wantLayer string
	}{
		{
			name:      "Go with bun ORM",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/uptrace/bun v1.2.0\n"},
			wantLayer: "bun",
		},
		{
			name:      "Go with gorm",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire gorm.io/gorm v1.25.0\n"},
			wantLayer: "gorm",
		},
		{
			name:      "Go with sqlx",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/jmoiron/sqlx v1.3.5\n"},
			wantLayer: "sqlx",
		},
		{
			name:      "Go with ent",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire entgo.io/ent v0.13.0\n"},
			wantLayer: "ent",
		},
		{
			name:      "Go with pgx",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/jackc/pgx/v5 v5.5.0\n"},
			wantLayer: "pgx",
		},
		{
			name:      "Go with no ORM returns none",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\ngo 1.21\n"},
			wantLayer: "none",
		},
		{
			name:      "Node.js with prisma",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"devDependencies":{"prisma":"^5.0.0"},"dependencies":{"@prisma/client":"^5.0.0"}}`},
			wantLayer: "prisma",
		},
		{
			name:      "Node.js with typeorm",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"typeorm":"^0.3.0"}}`},
			wantLayer: "typeorm",
		},
		{
			name:      "Node.js with drizzle",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"drizzle-orm":"^0.29.0"}}`},
			wantLayer: "drizzle",
		},
		{
			name:      "Node.js with mongoose",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"mongoose":"^8.0.0"}}`},
			wantLayer: "mongoose",
		},
		{
			name:      "Node.js with no ORM returns none",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"express":"^4.0.0"}}`},
			wantLayer: "none",
		},
		{
			name:      "Python with sqlalchemy",
			language:  "python",
			files:     map[string]string{"requirements.txt": "SQLAlchemy>=2.0\nfastapi\n"},
			wantLayer: "sqlalchemy",
		},
		{
			name:      "Python with django ORM",
			language:  "python",
			files:     map[string]string{"requirements.txt": "Django>=4.0\n"},
			wantLayer: "django-orm",
		},
		{
			name:      "Ruby always returns activerecord",
			language:  "ruby",
			files:     map[string]string{},
			wantLayer: "activerecord",
		},
		{
			name:      "flutter returns none",
			language:  "flutter",
			files:     map[string]string{},
			wantLayer: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := detectDataLayer(dir, tt.language)
			if got != tt.wantLayer {
				t.Errorf("detectDataLayer(%q) = %q, want %q", tt.language, got, tt.wantLayer)
			}
		})
	}
}

// ── detectHasQueue ────────────────────────────────────────────────────────────

func TestDetectHasQueue(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		files     map[string]string
		wantQueue bool
	}{
		{
			name:      "Go with rabbitmq amqp091-go",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/rabbitmq/amqp091-go v1.9.0\n"},
			wantQueue: true,
		},
		{
			name:      "Go with streadway/amqp",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/streadway/amqp v1.1.0\n"},
			wantQueue: true,
		},
		{
			name:      "Go with nats",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/nats-io/nats.go v1.34.0\n"},
			wantQueue: true,
		},
		{
			name:      "Go with kafka",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/confluentinc/confluent-kafka-go v1.9.2\n"},
			wantQueue: true,
		},
		{
			name:      "Go with no queue deps",
			language:  "go",
			files:     map[string]string{"go.mod": "module foo\n\nrequire github.com/go-chi/chi/v5 v5.0.12\n"},
			wantQueue: false,
		},
		{
			name:      "Node.js with amqplib",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"amqplib":"^0.10.0"}}`},
			wantQueue: true,
		},
		{
			name:      "Node.js with bull",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"bull":"^4.0.0"}}`},
			wantQueue: true,
		},
		{
			name:      "Node.js with bullmq",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"bullmq":"^5.0.0"}}`},
			wantQueue: true,
		},
		{
			name:      "Node.js with kafkajs",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"kafkajs":"^2.0.0"}}`},
			wantQueue: true,
		},
		{
			name:      "Node.js with nats",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"nats":"^2.0.0"}}`},
			wantQueue: true,
		},
		{
			name:      "Node.js with no queue deps",
			language:  "nodejs",
			files:     map[string]string{"package.json": `{"dependencies":{"express":"^4.0.0"}}`},
			wantQueue: false,
		},
		{
			name:      "Python with celery",
			language:  "python",
			files:     map[string]string{"requirements.txt": "celery>=5.0\n"},
			wantQueue: true,
		},
		{
			name:      "Python with pika",
			language:  "python",
			files:     map[string]string{"requirements.txt": "pika>=1.3\n"},
			wantQueue: true,
		},
		{
			name:      "Python with kombu",
			language:  "python",
			files:     map[string]string{"requirements.txt": "kombu>=5.0\n"},
			wantQueue: true,
		},
		{
			name:      "Python with no queue",
			language:  "python",
			files:     map[string]string{"requirements.txt": "fastapi\n"},
			wantQueue: false,
		},
		{
			name:      "rust always returns false",
			language:  "rust",
			files:     map[string]string{},
			wantQueue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := detectHasQueue(dir, tt.language)
			if got != tt.wantQueue {
				t.Errorf("detectHasQueue(%q) = %v, want %v", tt.language, got, tt.wantQueue)
			}
		})
	}
}

// ── detectIsInternal ────────────────────────────────────────────────────────────

func TestDetectIsInternal(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		wantInternal bool
	}{
		{
			name: "CLAUDE.md contains originating project",
			files: map[string]string{
				"CLAUDE.md": "# originating project Service\n\nThis is a originating project microservice.\n",
			},
			wantInternal: true,
		},
		{
			name: "docker-compose.yml contains internal (lowercase)",
			files: map[string]string{
				"docker-compose.yml": "version: '3'\nservices:\n  internal-api:\n    image: internal/api\n",
			},
			wantInternal: true,
		},
		{
			name: "docker-compose.yaml contains internal",
			files: map[string]string{
				"docker-compose.yaml": "services:\n  internal-worker:\n    image: myapp\n",
			},
			wantInternal: true,
		},
		{
			name: "compose.yml contains internal",
			files: map[string]string{
				"compose.yml": "services:\n  internal-db:\n    image: postgres\n",
			},
			wantInternal: true,
		},
		{
			name: "compose.yaml contains internal",
			files: map[string]string{
				"compose.yaml": "services:\n  internal-redis:\n    image: redis\n",
			},
			wantInternal: true,
		},
		{
			name: "no originating project markers",
			files: map[string]string{
				"CLAUDE.md":          "# My Service\n\nA generic service.\n",
				"docker-compose.yml": "services:\n  api:\n    image: myapp\n",
			},
			wantInternal: false,
		},
		{
			name:         "empty project returns false",
			files:        map[string]string{},
			wantInternal: false,
		},
		{
			name: "CLAUDE.md present but no originating project keyword",
			files: map[string]string{
				"CLAUDE.md": "# My App\n\nThis is not internal.\n",
			},
			wantInternal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := detectIsInternal(dir)
			if got != tt.wantInternal {
				t.Errorf("detectIsInternal() = %v, want %v", got, tt.wantInternal)
			}
		})
	}
}

// ── detectProjectType ─────────────────────────────────────────────────────────

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string // returns dir
		wantType string
	}{
		{
			name: "single go.mod at root is single-app",
			setup: func(t *testing.T) string {
				return makeTempProject(t, map[string]string{
					"go.mod": "module foo\n",
				})
			},
			wantType: "single-app",
		},
		{
			name: "two subdirs with go.mod each is monorepo",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = makeTempProject(t, map[string]string{})
				os.MkdirAll(filepath.Join(dir, "service-a"), 0755)
				os.MkdirAll(filepath.Join(dir, "service-b"), 0755)
				os.WriteFile(filepath.Join(dir, "service-a", "go.mod"), []byte("module a\n"), 0644)
				os.WriteFile(filepath.Join(dir, "service-b", "go.mod"), []byte("module b\n"), 0644)
				return dir
			},
			wantType: "monorepo",
		},
		{
			name: "node_modules subdir is skipped",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "node_modules", "foo"), 0755)
				os.WriteFile(filepath.Join(dir, "node_modules", "foo", "package.json"), []byte(`{"name":"foo"}`), 0644)
				return dir
			},
			wantType: "single-app",
		},
		{
			name: "hidden dirs are skipped",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, ".hidden-svc"), 0755)
				os.WriteFile(filepath.Join(dir, ".hidden-svc", "go.mod"), []byte("module hidden\n"), 0644)
				os.MkdirAll(filepath.Join(dir, ".another"), 0755)
				os.WriteFile(filepath.Join(dir, ".another", "go.mod"), []byte("module another\n"), 0644)
				return dir
			},
			wantType: "single-app",
		},
		{
			name: "vendor dir is skipped",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "vendor", "foo"), 0755)
				os.WriteFile(filepath.Join(dir, "vendor", "foo", "go.mod"), []byte("module foo\n"), 0644)
				os.MkdirAll(filepath.Join(dir, "vendor", "bar"), 0755)
				os.WriteFile(filepath.Join(dir, "vendor", "bar", "go.mod"), []byte("module bar\n"), 0644)
				return dir
			},
			wantType: "single-app",
		},
		{
			name: "mixed language monorepo: go + nodejs",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "api"), 0755)
				os.MkdirAll(filepath.Join(dir, "frontend"), 0755)
				os.WriteFile(filepath.Join(dir, "api", "go.mod"), []byte("module api\n"), 0644)
				os.WriteFile(filepath.Join(dir, "frontend", "package.json"), []byte(`{"name":"frontend"}`), 0644)
				return dir
			},
			wantType: "monorepo",
		},
		{
			name: "only one subproject is single-app",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "service-a"), 0755)
				os.WriteFile(filepath.Join(dir, "service-a", "go.mod"), []byte("module a\n"), 0644)
				return dir
			},
			wantType: "single-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			got := detectProjectType(dir)
			if got != tt.wantType {
				t.Errorf("detectProjectType() = %q, want %q", got, tt.wantType)
			}
		})
	}
}

// ── detectObservability ───────────────────────────────────────────────────────

func TestDetectObservability(t *testing.T) {
	tests := []struct {
		name     string
		language string
		files    map[string]string
		wantObs  []string // sorted for comparison
	}{
		{
			name:     "Go with zerolog + sentry + newrelic",
			language: "go",
			files: map[string]string{
				"go.mod": "module foo\n\nrequire (\n" +
					"\tgithub.com/rs/zerolog v1.31.0\n" +
					"\tgithub.com/getsentry/sentry-go v0.27.0\n" +
					"\tgithub.com/newrelic/go-agent/v3 v3.30.0\n)\n",
			},
			wantObs: []string{"newrelic", "sentry", "zerolog"},
		},
		{
			name:     "Go with zap",
			language: "go",
			files: map[string]string{
				"go.mod": "module foo\n\nrequire go.uber.org/zap v1.27.0\n",
			},
			wantObs: []string{"zap"},
		},
		{
			name:     "Go with opentelemetry",
			language: "go",
			files: map[string]string{
				"go.mod": "module foo\n\nrequire go.opentelemetry.io/otel v1.24.0\n",
			},
			wantObs: []string{"opentelemetry"},
		},
		{
			name:     "Go with no observability",
			language: "go",
			files: map[string]string{
				"go.mod": "module foo\n\ngo 1.21\n",
			},
			wantObs: nil,
		},
		{
			name:     "Node.js with pino + sentry + newrelic",
			language: "nodejs",
			files: map[string]string{
				"package.json": `{"dependencies":{"pino":"^8.0.0","@sentry/node":"^7.0.0","newrelic":"^11.0.0"}}`,
			},
			wantObs: []string{"newrelic", "pino", "sentry"},
		},
		{
			name:     "Node.js with winston",
			language: "nodejs",
			files: map[string]string{
				"package.json": `{"dependencies":{"winston":"^3.0.0"}}`,
			},
			wantObs: []string{"winston"},
		},
		{
			name:     "Python with sentry-sdk",
			language: "python",
			files: map[string]string{
				"requirements.txt": "sentry-sdk>=1.0\nfastapi\n",
			},
			wantObs: []string{"sentry"},
		},
		{
			name:     "Python with structlog",
			language: "python",
			files: map[string]string{
				"requirements.txt": "structlog>=23.0\n",
			},
			wantObs: []string{"structlog"},
		},
		{
			name:     "unknown language returns nil",
			language: "unknown",
			files:    map[string]string{},
			wantObs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := detectObservability(dir, tt.language)

			// Sort both for stable comparison
			sort.Strings(got)
			sort.Strings(tt.wantObs)

			if len(got) != len(tt.wantObs) {
				t.Errorf("detectObservability(%q) = %v, want %v", tt.language, got, tt.wantObs)
				return
			}
			for i := range got {
				if got[i] != tt.wantObs[i] {
					t.Errorf("detectObservability(%q)[%d] = %q, want %q", tt.language, i, got[i], tt.wantObs[i])
				}
			}
		})
	}
}

// ── RunDetection ──────────────────────────────────────────────────────────────

func TestRunDetection_GoProject(t *testing.T) {
	dir := makeTempProject(t, map[string]string{
		"go.mod": "module example.com/myapp\n\ngo 1.21\n\n" +
			"require (\n" +
			"\tgithub.com/go-chi/chi/v5 v5.0.12\n" +
			"\tgithub.com/uptrace/bun v1.2.0\n" +
			"\tgithub.com/onsi/ginkgo/v2 v2.15.0\n" +
			"\tgithub.com/rabbitmq/amqp091-go v1.9.0\n" +
			"\tgithub.com/rs/zerolog v1.31.0\n" +
			")\n",
		"CLAUDE.md": "# originating project Service\n",
	})
	makeTempDirs(t, dir, "cmd", "internal")

	result := RunDetection(dir)

	if result.Language != "go" {
		t.Errorf("Language = %q, want %q", result.Language, "go")
	}
	if result.ArchStyle != "layered" {
		t.Errorf("ArchStyle = %q, want %q", result.ArchStyle, "layered")
	}
	if result.TestingFramework != "ginkgo" {
		t.Errorf("TestingFramework = %q, want %q", result.TestingFramework, "ginkgo")
	}
	if result.DataLayer != "bun" {
		t.Errorf("DataLayer = %q, want %q", result.DataLayer, "bun")
	}
	if !result.HasQueue {
		t.Error("HasQueue should be true")
	}
	if !result.IsInternal {
		t.Error("IsInternal should be true")
	}
}

func TestRunDetection_UnknownProject(t *testing.T) {
	dir := t.TempDir()
	result := RunDetection(dir)

	if result.Language != "unknown" {
		t.Errorf("Language = %q, want %q", result.Language, "unknown")
	}
	if result.HasQueue {
		t.Error("HasQueue should be false for unknown project")
	}
	if result.IsInternal {
		t.Error("IsInternal should be false for unknown project")
	}
}

// ── extractGoModVersion ───────────────────────────────────────────────────────

func TestExtractGoModVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		module  string
		want    string
	}{
		{
			name:    "chi v5 path suffix",
			content: "require github.com/go-chi/chi/v5 v5.0.12\n",
			module:  "github.com/go-chi/chi",
			want:    "v5",
		},
		{
			name:    "echo v4 path suffix",
			content: "require github.com/labstack/echo/v4 v4.12.0\n",
			module:  "github.com/labstack/echo",
			want:    "v4",
		},
		{
			name:    "module not in content returns empty",
			content: "require github.com/go-chi/chi/v5 v5.0.12\n",
			module:  "github.com/gin-gonic/gin",
			want:    "",
		},
		{
			name:    "module without version suffix returns v1",
			content: "require github.com/gin-gonic/gin v1.9.1\n",
			module:  "github.com/gin-gonic/gin",
			want:    "v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGoModVersion(tt.content, tt.module)
			if got != tt.want {
				t.Errorf("extractGoModVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── cleanVersion ──────────────────────────────────────────────────────────────

func TestCleanVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"^14.0.0", "14.0.0"},
		{"~4.18.0", "4.18.0"},
		{">=1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanVersion(tt.input)
			if got != tt.want {
				t.Errorf("cleanVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
