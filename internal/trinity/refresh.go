package trinity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/staleness"
)

// RefreshOpts configures the trinity refresh command.
type RefreshOpts struct {
	ProjectDir   string
	MaxAgeMonths float64
	Topics       []string
	DryRun       bool
	StaleOnly    bool
}

// staleDoc represents a single stale knowledge document.
type staleDoc struct {
	Filename string
	DateStr  string
	AgeStr   string
}

// staleGroup groups stale docs by their parent directory.
type staleGroup struct {
	DirName string
	Docs    []staleDoc
}

// RunRefresh detects stale knowledge docs and guides the user through refreshing them.
func RunRefresh(opts RefreshOpts) {
	root, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		root = opts.ProjectDir
	}
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}
	relRoot := cli.PathToHome(root)

	header := "trinity refresh"
	if opts.DryRun {
		header += cli.Dim(" (dry run)")
	}
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan(header+" — "+relRoot)))

	// ─── 1. Find knowledge directories ────────────────────────────────────────
	knowledgeDir := filepath.Join(root, ".claude", "knowledge")

	if !config.IsDir(knowledgeDir) {
		fmt.Printf("  %s\n\n", cli.Red("✗  .claude/knowledge/ not found"))
		os.Exit(1)
	}

	entries, err := os.ReadDir(knowledgeDir)
	if err != nil {
		fmt.Printf("  %s\n\n", cli.Red(fmt.Sprintf("✗  %v", err)))
		os.Exit(1)
	}

	var subdirNames []string
	for _, e := range entries {
		if e.IsDir() {
			subdirNames = append(subdirNames, e.Name())
		}
	}
	sort.Strings(subdirNames)

	if len(subdirNames) == 0 {
		fmt.Printf("  %s\n\n", cli.Yellow("No knowledge subdirectories found"))
		os.Exit(0)
	}

	// ─── 2. Scan for stale docs ───────────────────────────────────────────────
	var allStale []staleGroup

	for _, name := range subdirNames {
		dirPath := filepath.Join(knowledgeDir, name)
		docs := findStaleDocs(dirPath, opts.MaxAgeMonths, opts.Topics)
		if len(docs) > 0 {
			allStale = append(allStale, staleGroup{DirName: name, Docs: docs})
		}
	}

	// ─── 3. Report ────────────────────────────────────────────────────────────
	if len(allStale) == 0 {
		if len(opts.Topics) > 0 {
			fmt.Printf("  %s\n\n", cli.Green("✓  No matching stale topics found"))
		} else {
			fmt.Printf("  %s\n\n", cli.Green(fmt.Sprintf("✓  All knowledge docs are fresh (threshold: %.0f months)", opts.MaxAgeMonths)))
		}
		os.Exit(0)
	}

	// Print stale docs grouped by directory
	totalStale := 0

	for _, group := range allStale {
		fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf(".claude/knowledge/%s/", group.DirName)))
		for _, doc := range group.Docs {
			fmt.Printf("    %s%s\n", cli.Yellow("⚠  "+doc.Filename), cli.Dim("  "+doc.DateStr+", "+doc.AgeStr))
			totalStale++
		}
		fmt.Println()
	}

	label := "doc"
	if totalStale > 1 {
		label = "docs"
	}
	fmt.Printf("  %s\n\n", cli.Bold(fmt.Sprintf("%d stale %s found", totalStale, label)))

	// ─── 4. Stale topic slugs ────────────────────────────────────────────────
	var staleTopics []string
	for _, group := range allStale {
		for _, doc := range group.Docs {
			staleTopics = append(staleTopics, strings.TrimSuffix(doc.Filename, ".md"))
		}
	}

	cli.PrintDim(fmt.Sprintf("Stale topics: %s", strings.Join(staleTopics, ", ")))
	fmt.Println()

	// ─── 5. Print manual workflow ────────────────────────────────────────────
	{
		cli.PrintDim("Manual refresh workflow:")
		fmt.Println()

		// Load .trinity.json for context if available
		cfg := LoadTrinityConfig(root)

		if cfg != nil {
			for _, group := range allStale {
				// Find matching stack
				var matchedStack *SyncPair
				for i := range cfg.Stacks {
					if strings.Contains(cfg.Stacks[i].To, group.DirName) {
						matchedStack = &cfg.Stacks[i]
						break
					}
				}

				if matchedStack != nil {
					cli.PrintDim(fmt.Sprintf("# Re-research %s:", matchedStack.Name))
					cli.PrintWhite("oracle research")
					cli.PrintDim("# Then sync:")
					cli.PrintWhite(fmt.Sprintf("trinity sync --from %s --to %s --path %s", matchedStack.From, matchedStack.To, relRoot))
				} else {
					cli.PrintDim(fmt.Sprintf("# Re-research %s:", group.DirName))
					cli.PrintWhite("oracle research")
					cli.PrintDim(fmt.Sprintf("# Then sync results to .claude/knowledge/%s/", group.DirName))
				}
				fmt.Println()
			}
		} else {
			cli.PrintWhite("1. oracle research              # re-research the stack")
			cli.PrintWhite("2. trinity sync --from <output> --to <knowledge-dir>")
			fmt.Println()
		}
	}
}

// ─── Staleness scanner ───────────────────────────────────────────────────────

func findStaleDocs(dirPath string, maxAgeMonths float64, topicFilter []string) []staleDoc {
	mdFiles, err := config.ListMDFiles(dirPath)
	if err != nil {
		return nil
	}

	var stale []staleDoc

	for _, filename := range mdFiles {
		// Filter by topic if specified
		if len(topicFilter) > 0 {
			slug := strings.TrimSuffix(filename, ".md")
			matched := false
			for _, t := range topicFilter {
				if strings.Contains(slug, t) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		filePath := filepath.Join(dirPath, filename)
		content, err := config.ReadFileString(filePath)
		if err != nil {
			continue
		}

		created, ok := staleness.ParseCreatedDate(content)
		if !ok {
			// No date = can't determine freshness, treat as potentially stale
			stale = append(stale, staleDoc{
				Filename: filename,
				DateStr:  "?",
				AgeStr:   "no date found",
			})
			continue
		}

		age := staleness.MonthsAgo(created)
		if age >= maxAgeMonths {
			stale = append(stale, staleDoc{
				Filename: filename,
				DateStr:  created.Format("2006-01-02"),
				AgeStr:   staleness.FormatAge(created),
			})
		}
	}

	return stale
}

