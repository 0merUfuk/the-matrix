package trinity

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/0merUfuk/the-matrix/internal/neo"
)

func TestExpectedAgentsMatchNeoContract(t *testing.T) {
	want := []string{
		"manager.md", "developer.md", "tester.md", "reviewer.md",
		"strategist.md", "security-reviewer.md",
	}
	if !reflect.DeepEqual(ExpectedAgents, want) {
		t.Errorf("ExpectedAgents = %v, want Neo-generated contract %v", ExpectedAgents, want)
	}
}

func TestExpectedSkillsMatchNeoContract(t *testing.T) {
	want := []string{
		"audit", "commit", "continue", "dep-audit", "doublecheck",
		"fix", "issue", "owasp-review", "secret-scan", "security-scan",
	}
	if !reflect.DeepEqual(ExpectedSkills, want) {
		t.Errorf("ExpectedSkills = %v, want Neo-generated contract %v", ExpectedSkills, want)
	}
}

func TestTrinityHealthContractHelper(t *testing.T) {
	if os.Getenv("TRINITY_HEALTH_CONTRACT_HELPER") != "1" {
		return
	}
	RunHealth(HealthOpts{ProjectDir: os.Getenv("TRINITY_HEALTH_CONTRACT_PATH")})
}

func TestFreshNeoEcosystemHasNoMissingAgentOrSkillWarnings(t *testing.T) {
	dir := t.TempDir()
	profile := &neo.ProjectProfile{
		ProjectName: "contract-fixture",
		ProjectType: "single-app",
		TeamSize:    "solo",
		CreatedDate: "2026-07-29",
		OutputPath:  dir,
		Stacks: []neo.StackProfile{
			{
				Name:             "go-service",
				Language:         "go",
				Framework:        "standard library",
				ArchStyle:        "layered",
				TestingFramework: "testing",
				DataLayer:        "none",
			},
		},
	}
	if _, err := neo.GenerateEcosystem(profile, dir); err != nil {
		t.Fatalf("GenerateEcosystem: %v", err)
	}

	output, err := runTrinityHealthContractSubprocess(t, dir)
	if err != nil {
		t.Fatalf("trinity health exited unexpectedly: %v\n%s", err, output)
	}

	for _, line := range strings.Split(string(output), "\n") {
		if (strings.Contains(line, ".claude/agents/") || strings.Contains(line, ".claude/skills/")) && strings.Contains(strings.ToLower(line), "missing") {
			t.Errorf("fresh Neo ecosystem produced a missing agent/skill warning: %s", line)
		}
	}
}

func runTrinityHealthContractSubprocess(t *testing.T, dir string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run", "^TestTrinityHealthContractHelper$")
	cmd.Env = append(os.Environ(),
		"TRINITY_HEALTH_CONTRACT_HELPER=1",
		"TRINITY_HEALTH_CONTRACT_PATH="+dir,
	)
	return cmd.CombinedOutput()
}
