package neo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/staleness"
)

// RunStatus shows a dashboard view of the entire ecosystem.
func RunStatus(projectDir string) {
	root, _ := filepath.Abs(projectDir)
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}

	theme := cli.GetTheme()

	// Load .neo.json
	neoPath := filepath.Join(root, ".neo.json")
	if !config.FileExists(neoPath) {
		fmt.Fprintln(os.Stderr, theme.ErrorBox("No .neo.json found", "Run `neo init` to create one."))
		os.Exit(1)
	}

	profile, err := config.ReadJSON[ProjectProfile](neoPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, theme.ErrorBox(fmt.Sprintf("Invalid .neo.json: %v", err)))
		os.Exit(1)
	}

	fmt.Print(theme.Banner("neo status", profile.ProjectName))

	// Project info
	internalProjectLabel := "no"
	if profile.IsInternal {
		internalProjectLabel = "yes"
	}
	fmt.Println(theme.KeyValue([]cli.KV{
		{Key: "Project", Value: fmt.Sprintf("%s  |  %s  |  %s", profile.ProjectType, profile.TeamSize, profile.CreatedDate)},
		{Key: "originating project", Value: internalProjectLabel},
	}))
	fmt.Println()

	// Knowledge freshness per stack
	knowledgeDir := filepath.Join(root, ".claude", "knowledge")
	if config.IsDir(knowledgeDir) {
		fmt.Println(theme.SectionHeader("Knowledge"))
		for _, stack := range profile.Stacks {
			stackDir := filepath.Join(knowledgeDir, stack.Name)
			if !config.IsDir(stackDir) {
				fmt.Println(theme.StatusLine(cli.Fail, fmt.Sprintf("%-20s", stack.Name), "not found"))
				continue
			}

			files, _ := config.ListMDFiles(stackDir)
			docCount := len(files)

			// Check freshness
			freshness := "unknown"
			allFresh := true
			for _, f := range files {
				content, _ := config.ReadFileString(filepath.Join(stackDir, f))
				created, ok := staleness.ParseCreatedDate(content)
				if ok && staleness.IsStale(created, 6) {
					allFresh = false
					freshness = fmt.Sprintf("stale (%s)", created.Format("2006-01-02"))
					break
				}
			}
			if allFresh && docCount > 0 {
				freshness = "fresh"
			}

			relPath := fmt.Sprintf(".claude/knowledge/%s/", stack.Name)
			level := cli.Pass
			if !allFresh {
				level = cli.Warn
			}
			detail := fmt.Sprintf("%-35s %d docs   %s", relPath, docCount, freshness)
			fmt.Println(theme.StatusLine(level, fmt.Sprintf("%-20s", stack.Name), detail))
		}
		fmt.Println()
	}

	// Services (if multi-repo)
	if len(profile.Services) > 0 {
		fmt.Println(theme.SectionHeader("Services"))
		for _, svc := range profile.Services {
			svcDir := filepath.Join(root, svc.Name)
			if config.IsDir(svcDir) && config.FileExists(filepath.Join(svcDir, ".autonomous/loop.sh")) {
				fmt.Println(theme.StatusLine(cli.Pass, fmt.Sprintf("%-25s", svc.Name), "scaffolded"))
			} else {
				fmt.Println(theme.StatusLine(cli.Fail, fmt.Sprintf("%-25s", svc.Name), "not scaffolded"))
			}
		}
		fmt.Println()
	}

	// Trinity log
	logPath := filepath.Join(root, ".claude", "trinity-log.md")
	if config.FileExists(logPath) {
		content, err := config.ReadFileString(logPath)
		if err == nil {
			lines := strings.Split(strings.TrimSpace(content), "\n")
			// Find last non-empty, non-header line
			for i := len(lines) - 1; i >= 0; i-- {
				line := strings.TrimSpace(lines[i])
				if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ">") {
					fmt.Println(theme.KeyValue([]cli.KV{{Key: "Last sync", Value: theme.Muted.Render(line)}}))
					break
				}
			}
		}
	} else {
		fmt.Println(theme.KeyValue([]cli.KV{{Key: "Last sync", Value: theme.Muted.Render("never")}}))
	}

	fmt.Printf("\n  %s\n\n", theme.Muted.Render("Run: neo update to refresh stale knowledge"))
}
