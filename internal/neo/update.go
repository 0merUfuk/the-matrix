package neo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/staleness"
	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// RunUpdate checks for stale knowledge and guides refresh.
func RunUpdate(projectDir string) {
	root, _ := filepath.Abs(projectDir)
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}

	// Load .neo.json
	neoPath := filepath.Join(root, ".neo.json")
	if !config.FileExists(neoPath) {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red("No .neo.json found. Run neo init first."))
		os.Exit(1)
	}

	profile, err := config.ReadJSON[ProjectProfile](neoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Invalid .neo.json: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("neo update — "+profile.ProjectName)))

	// Check each stack's knowledge freshness
	knowledgeDir := filepath.Join(root, ".claude", "knowledge")
	var staleStacks []string

	for _, stack := range profile.Stacks {
		stackDir := filepath.Join(knowledgeDir, stack.Name)
		if !config.IsDir(stackDir) {
			fmt.Printf("  %s  %s — %s\n", cli.Red("✗"), stack.Name, cli.Dim("knowledge not found"))
			staleStacks = append(staleStacks, stack.Name)
			continue
		}

		files, _ := config.ListMDFiles(stackDir)
		isStale := false
		for _, f := range files {
			content, _ := config.ReadFileString(filepath.Join(stackDir, f))
			created, ok := staleness.ParseCreatedDate(content)
			if ok && staleness.IsStale(created, 6) {
				isStale = true
				break
			}
		}

		if isStale {
			fmt.Printf("  %s  %s — %s\n", cli.Yellow("⚠"), stack.Name, cli.Yellow("stale"))
			staleStacks = append(staleStacks, stack.Name)
		} else if len(files) > 0 {
			fmt.Printf("  %s  %s — %s\n", cli.Green("✓"), stack.Name, cli.Green("fresh"))
		} else {
			fmt.Printf("  %s  %s — %s\n", cli.Red("✗"), stack.Name, cli.Dim("empty"))
			staleStacks = append(staleStacks, stack.Name)
		}
	}

	fmt.Println()

	if len(staleStacks) == 0 {
		fmt.Printf("  %s\n\n", cli.Green("All knowledge is fresh. Nothing to update."))
		return
	}

	fmt.Printf("  %s\n\n", cli.Bold(fmt.Sprintf("%d stack(s) need refresh: %s", len(staleStacks), cli.Yellow(fmt.Sprintf("%v", staleStacks)))))

	// Ask to refresh
	doRefresh, err := wizard.AskConfirm("Refresh stale knowledge?", true)
	if err != nil || !doRefresh {
		fmt.Println("  Skipped.")
		return
	}

	// Print manual workflow (oracle research + trinity sync)
	fmt.Println()
	cli.PrintDim("Manual refresh workflow:")
	for _, stackName := range staleStacks {
		fmt.Println()
		cli.PrintDim(fmt.Sprintf("# Refresh %s:", stackName))
		cli.PrintWhite(fmt.Sprintf("oracle research   # Re-research %s stack", stackName))
		cli.PrintWhite(fmt.Sprintf("trinity sync --from output/%s/ --to .claude/knowledge/%s/", stackName, stackName))
	}
	fmt.Println()
}
