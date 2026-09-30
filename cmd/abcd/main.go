// Command abcd is the CLI front door to the abcd engine. It is a thin shell:
// all behaviour lives in internal/core, surfaced here via internal/surface/cli.
package main

import (
	"os"

	"github.com/intentdriven/abcd/internal/adapter/gitleaks"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/surface/cli"
)

func main() {
	wire()
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}

// wire is the composition root: it registers the adapters the core reaches
// through a seam it declares. The gitleaks augmenter is one: every scanner the
// core builds picks it up, and it reads the repository's own opt-in
// (.abcd/config/gitleaks.json), so a repository that did not opt in pays
// nothing (iss-2608291814575788).
func wire() {
	scanner.SetDefaultAugmenter(gitleaks.NewAugmenter)
}
