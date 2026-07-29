package neo

import (
	"fmt"
	"os"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/registry"
	"github.com/0merUfuk/the-matrix/internal/validation"
)

// RunRegistryList fetches the registry index and displays all available packs.
func RunRegistryList() {
	cli.PrintHeader("neo", "registry -- knowledge packs")

	var index *registry.RegistryIndex

	err := cli.WithSpinner("Fetching registry index...", func() error {
		var fetchErr error
		index, fetchErr = registry.Load()
		return fetchErr
	})
	if err != nil {
		cli.PrintError(fmt.Sprintf("Failed to load registry: %v", err))
		os.Exit(1)
	}

	if len(index.Packs) == 0 {
		cli.PrintDim("No knowledge packs available.")
		fmt.Println()
		return
	}

	printPackTable(index.Packs)
	fmt.Println()
	cli.PrintDim(fmt.Sprintf("  %d packs available. Use: neo registry pull <name> --to <dir>", len(index.Packs)))
	fmt.Println()
}

// RunRegistrySearch filters the registry by a query and displays matching packs.
func RunRegistrySearch(query string) {
	cli.PrintHeader("neo", "registry -- search")

	var index *registry.RegistryIndex

	err := cli.WithSpinner("Fetching registry index...", func() error {
		var fetchErr error
		index, fetchErr = registry.Load()
		return fetchErr
	})
	if err != nil {
		cli.PrintError(fmt.Sprintf("Failed to load registry: %v", err))
		os.Exit(1)
	}

	results := registry.Search(index, query)
	if len(results) == 0 {
		cli.PrintDim(fmt.Sprintf("No packs matching %q.", query))
		fmt.Println()
		return
	}

	printPackTable(results)
	fmt.Println()
	cli.PrintDim(fmt.Sprintf("  %d packs matching %q.", len(results), query))
	fmt.Println()
}

// RunRegistryInfo displays full details for a named pack.
func RunRegistryInfo(packName string) {
	cli.PrintHeader("neo", "registry -- pack info")

	var detail *registry.PackDetail

	err := cli.WithSpinner(fmt.Sprintf("Fetching %s...", packName), func() error {
		var fetchErr error
		detail, fetchErr = registry.Info(packName)
		return fetchErr
	})
	if err != nil {
		cli.PrintError(fmt.Sprintf("Failed to fetch pack info: %v", err))
		os.Exit(1)
	}

	// Pack header.
	fmt.Printf("  %s\n\n", cli.Bold(detail.DisplayName))

	// Metadata.
	fmt.Printf("  %-14s %s\n", cli.Dim("Name:"), detail.Name)
	fmt.Printf("  %-14s %s\n", cli.Dim("Language:"), detail.Language)
	fmt.Printf("  %-14s %s\n", cli.Dim("Framework:"), detail.Framework)
	fmt.Printf("  %-14s %s\n", cli.Dim("Version:"), detail.Version)
	fmt.Printf("  %-14s %s\n", cli.Dim("Arch Style:"), detail.ArchStyle)
	fmt.Printf("  %-14s %s\n", cli.Dim("Score:"), fmt.Sprintf("%d/100", detail.QualityScore))
	fmt.Printf("  %-14s %s\n", cli.Dim("Created:"), detail.CreatedDate)
	fmt.Printf("  %-14s %s\n", cli.Dim("Refreshed:"), detail.RefreshedDate)
	fmt.Printf("  %-14s %s\n", cli.Dim("Researched By:"), detail.ResearchedBy)
	if detail.GoldStandard != "" {
		fmt.Printf("  %-14s %s\n", cli.Dim("Gold Std:"), detail.GoldStandard)
	}
	fmt.Println()

	// Description.
	if detail.Description != "" {
		fmt.Printf("  %s\n\n", detail.Description)
	}

	// Topics table.
	if len(detail.Topics) > 0 {
		fmt.Printf("  %s (%d)\n\n", cli.Bold("Topics"), len(detail.Topics))
		fmt.Printf("  %-35s %6s %6s\n", cli.Dim("File"), cli.Dim("Lines"), cli.Dim("T1"))
		fmt.Printf("  %-35s %6s %6s\n", cli.Dim(strings.Repeat("-", 35)), cli.Dim("-----"), cli.Dim("---"))
		for _, topic := range detail.Topics {
			fmt.Printf("  %-35s %6d %6d\n", topic.File, topic.Lines, topic.Tier1Sources)
		}
		fmt.Println()
	}

	// Score verification.
	computed := registry.ComputeScore(*detail)
	if computed == detail.QualityScore {
		cli.PrintLine(cli.Pass, "Score verified", fmt.Sprintf("%d/100", computed))
	} else {
		cli.PrintLine(cli.Warn, "Score mismatch", fmt.Sprintf("claimed %d, computed %d", detail.QualityScore, computed))
	}
	fmt.Println()
}

// RunRegistryPull downloads a pack's knowledge docs to the target directory.
func RunRegistryPull(packName, targetDir string) {
	cli.PrintHeader("neo", "registry -- pull")

	// Validation function uses the shared knowledge-document validator.
	validateFn := func(filename, content string) error {
		errs := validation.ValidateInjectDoc(filename, content)
		if len(errs) > 0 {
			return fmt.Errorf("%s", strings.Join(errs, "; "))
		}
		return nil
	}

	fmt.Printf("  Pack:   %s\n", packName)
	fmt.Printf("  Target: %s\n\n", cli.PathToHome(targetDir))

	err := cli.WithSpinner(fmt.Sprintf("Pulling %s...", packName), func() error {
		return registry.Pull(packName, targetDir, validateFn)
	})
	if err != nil {
		cli.PrintError(fmt.Sprintf("Pull failed: %v", err))
		os.Exit(1)
	}

	// Count downloaded files.
	entries, _ := os.ReadDir(targetDir)
	docCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			docCount++
		}
	}

	cli.PrintSuccess(fmt.Sprintf("Pulled %d docs to %s", docCount, cli.PathToHome(targetDir)))
	fmt.Println()
	cli.PrintDim("  Docs have been validated with oracle's quality checks.")
	cli.PrintDim("  Run: neo doctor --path <project> to verify ecosystem integrity.")
	fmt.Println()
}

// printPackTable prints a formatted table of pack metadata.
func printPackTable(packs []registry.PackMeta) {
	// Header.
	fmt.Printf("  %-20s %-10s %-12s %5s %5s  %s\n",
		cli.Dim("Name"), cli.Dim("Language"), cli.Dim("Framework"),
		cli.Dim("Score"), cli.Dim("Docs"), cli.Dim("Refreshed"))
	fmt.Printf("  %-20s %-10s %-12s %5s %5s  %s\n",
		cli.Dim(strings.Repeat("-", 20)),
		cli.Dim(strings.Repeat("-", 10)),
		cli.Dim(strings.Repeat("-", 12)),
		cli.Dim("-----"),
		cli.Dim("-----"),
		cli.Dim(strings.Repeat("-", 10)))

	for _, p := range packs {
		scoreStr := fmt.Sprintf("%d", p.QualityScore)
		docsStr := fmt.Sprintf("%d", p.TopicCount)
		fmt.Printf("  %-20s %-10s %-12s %5s %5s  %s\n",
			p.Name, p.Language, p.Framework, scoreStr, docsStr, p.RefreshedDate)
	}
}
