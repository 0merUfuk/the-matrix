// Command matrix opens The Matrix: a local web app that runs neo, oracle,
// morp and trinity for people who do not use the command line.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0merUfuk/the-matrix/internal/matrixapp"
	"github.com/spf13/cobra"
)

var version = "1.9.0-dev"

func main() {
	var path, addr string
	var maxAgeMonths float64
	var noOpen bool

	cmd := &cobra.Command{
		Use:   "matrix",
		Short: "The Matrix: set up and look after agent ecosystems from your browser",
		Long: `matrix opens The Matrix in your browser. Pick a project folder, set up its AI
agents, check their health and keep their knowledge fresh, without using the
command line. It runs only on this computer and changes files only after you
confirm.`,
		Example: `  # Open The Matrix
  matrix

  # Open it with a project already in the list
  matrix --path ~/code/my-app

  # Run in the background (for example as a Homebrew service) without opening a browser
  matrix --no-open`,
		Version: version,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return matrixapp.Serve(ctx, matrixapp.Options{
				ProjectDir:   path,
				Addr:         addr,
				MaxAgeMonths: maxAgeMonths,
				Version:      version,
				Open:         !noOpen,
			})
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Add this project folder to the list")
	cmd.Flags().StringVar(&addr, "addr", matrixapp.DefaultAddr, "Address on this computer to listen on")
	cmd.Flags().Float64Var(&maxAgeMonths, "max-age-months", 6, "Months before knowledge counts as out of date")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Do not open the browser")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
