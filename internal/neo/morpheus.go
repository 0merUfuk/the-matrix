package neo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// RunMorpheusIntegration optionally scaffolds services using morpheus after neo init.
// It is a no-op for non-multi-repo-microservices projects, when morpheus is not
// installed, or when there are no services in the profile. All failure modes are
// handled internally (warnings printed, execution continues) — the function never
// returns an error.
func RunMorpheusIntegration(profile *ProjectProfile, outputDir string) {
	// Only applicable for multi-repo-microservices projects.
	if profile.ProjectType != "multi-repo-microservices" {
		return
	}

	// Nothing to scaffold if no services were defined.
	if len(profile.Services) == 0 {
		return
	}

	// Check if morp binary is available on PATH.
	if _, err := exec.LookPath("morp"); err != nil {
		fmt.Printf("\n  %s\n", cli.Dim("morp not found. Install it to scaffold services: brew install 0merUfuk/thematrix/morp"))
		return
	}

	// List services and prompt.
	fmt.Printf("\n  Found %d services in project profile:\n", len(profile.Services))
	for i, svc := range profile.Services {
		fmt.Printf("    %d. %s\n", i+1, svc.Name)
	}
	fmt.Println()

	confirm, err := wizard.AskConfirm("Run morp init for each service?", true)
	if err != nil {
		// User interrupted (ctrl-c) — not a fatal error.
		return
	}
	if !confirm {
		cli.PrintDim("Run 'morp init' in each service directory to scaffold manually.")
		return
	}

	contextPath := filepath.Join(outputDir, ".neo.json")
	var failed []string

	for _, svc := range profile.Services {
		if svc.Name == "" {
			cli.PrintWarning("skipping service with empty name")
			failed = append(failed, "(unnamed)")
			continue
		}

		svcOutputDir := filepath.Join(outputDir, svc.Name)

		fmt.Printf("\n  %s\n\n",
			cli.Bold(cli.Cyan(fmt.Sprintf("── morp init: %s ──", svc.Name))))

		cmd := exec.Command("morp", "init",
			"--context", contextPath,
			"--output-dir", svcOutputDir,
			"--service-name", svc.Name,
		)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if runErr := cmd.Run(); runErr != nil {
			cli.PrintWarning(fmt.Sprintf("%s failed: %v", svc.Name, runErr))
			failed = append(failed, svc.Name)
		} else {
			cli.PrintSuccess(fmt.Sprintf("%s scaffolded", svc.Name))
		}
	}

	// Print summary.
	fmt.Println()
	total := len(profile.Services)
	failCount := len(failed)

	if failCount == 0 {
		cli.PrintSuccess(fmt.Sprintf("All %d services scaffolded successfully.", total))
	} else {
		succeeded := total - failCount
		cli.PrintWarning(fmt.Sprintf("%d of %d services failed: %v", failCount, total, failed))
		if succeeded > 0 {
			cli.PrintSuccess(fmt.Sprintf("%d of %d services scaffolded successfully.", succeeded, total))
		}
	}

}
