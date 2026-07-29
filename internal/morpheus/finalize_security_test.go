package morpheus

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFinalizeSecurityHelper(t *testing.T) {
	if os.Getenv("MORPHEUS_FINALIZE_SECURITY_HELPER") != "1" {
		return
	}

	RunFinalize()
}

func TestFinalizeSecurityLatestReviewerWithoutSecurityRefuses(t *testing.T) {
	projectDir, marker := createFinalizeSecurityProject(t, map[string]string{
		"cycle-01-review.md": "## Verdict: APPROVED\n",
	})

	output, err := runFinalizeSecuritySubprocess(t, projectDir)
	assertFinalizeSecurityRefused(t, err, output, marker)
}

func TestFinalizeSecurityDualApprovalProceeds(t *testing.T) {
	projectDir, marker := createFinalizeSecurityProject(t, map[string]string{
		"cycle-01-review.md":   "## Verdict: APPROVED\n",
		"cycle-01-security.md": "## Verdict: SECURITY_APPROVED\n",
	})

	output, err := runFinalizeSecuritySubprocess(t, projectDir)
	if err != nil {
		t.Fatalf("RunFinalize refused the latest cycle with both approvals: %v\n%s", err, output)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("RunFinalize did not run the finalizer after dual approval: %v\n%s", err, output)
	}
}

func TestFinalizeSecurityBlockedRefuses(t *testing.T) {
	projectDir, marker := createFinalizeSecurityProject(t, map[string]string{
		"cycle-01-review.md":   "## Verdict: APPROVED\n",
		"cycle-01-security.md": "## Verdict: SECURITY_BLOCKED\n",
	})

	output, err := runFinalizeSecuritySubprocess(t, projectDir)
	assertFinalizeSecurityRefused(t, err, output, marker)
}

func TestFinalizeSecurityLatestCycleBlockedRefuses(t *testing.T) {
	projectDir, marker := createFinalizeSecurityProject(t, map[string]string{
		"cycle-01-review.md":   "## Verdict: APPROVED\n",
		"cycle-01-security.md": "## Verdict: SECURITY_APPROVED\n",
		"cycle-02-review.md":   "## Verdict: APPROVED\n",
		"cycle-02-security.md": "## Verdict: SECURITY_BLOCKED\n",
	})

	output, err := runFinalizeSecuritySubprocess(t, projectDir)
	assertFinalizeSecurityRefused(t, err, output, marker)
}

func createFinalizeSecurityProject(t *testing.T, cycles map[string]string) (string, string) {
	t.Helper()

	projectDir := t.TempDir()
	cyclesDir := filepath.Join(projectDir, "tasks", "cycles")
	if err := os.MkdirAll(cyclesDir, 0755); err != nil {
		t.Fatalf("create cycles directory: %v", err)
	}
	for name, content := range cycles {
		if err := os.WriteFile(filepath.Join(cyclesDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write cycle file %s: %v", name, err)
		}
	}

	finalizerDir := filepath.Join(projectDir, ".autonomous")
	if err := os.MkdirAll(finalizerDir, 0755); err != nil {
		t.Fatalf("create finalizer directory: %v", err)
	}
	marker := filepath.Join(projectDir, ".finalizer-ran")
	finalizer := "#!/bin/bash\nset -e\nprintf 'ran\\n' > .finalizer-ran\n"
	if err := os.WriteFile(filepath.Join(finalizerDir, "finalizer.sh"), []byte(finalizer), 0755); err != nil {
		t.Fatalf("write finalizer: %v", err)
	}

	return projectDir, marker
}

func runFinalizeSecuritySubprocess(t *testing.T, projectDir string) ([]byte, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run", "^TestRunFinalizeSecurityHelper$")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "MORPHEUS_FINALIZE_SECURITY_HELPER=1")
	return cmd.CombinedOutput()
}

func assertFinalizeSecurityRefused(t *testing.T, err error, output []byte, marker string) {
	t.Helper()

	if err == nil {
		t.Fatalf("RunFinalize succeeded without latest-cycle dual approval:\n%s", output)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("RunFinalize executed the finalizer without latest-cycle dual approval: %v\n%s", statErr, output)
	}
	if !strings.Contains(string(output), "both reviewer APPROVED and security SECURITY_APPROVED") {
		t.Fatalf("RunFinalize did not explain the dual-approval requirement:\n%s", output)
	}
}
