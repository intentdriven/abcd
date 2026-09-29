// Command scaffold-render writes every profile of the scaffolded release
// workflows (scaffold.AuditProfiles) into a directory, one
// <profile>/.github/workflows/ tree each, for CI's zizmor job to audit
// (iss-2609251939472371). The committed workflows are the abcd profile only,
// so without it the profiles a managed repository receives were held to the
// in-repo audit's two finding classes and never to zizmor's pinning,
// permission and credential audits.
//
// Usage: go run ./cmd/scaffold-render <dir>. The directory is the caller's
// scratch space; nothing outside it is written. Exit 0 written, 1 a fault,
// 2 a usage error.
package main

import (
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: scaffold-render <dir>")
		os.Exit(2)
	}
	written, err := scaffold.WriteAuditProfiles(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "scaffold-render:", err)
		os.Exit(1)
	}
	fmt.Printf("scaffold-render: wrote %d workflow file(s) for %d profile(s)\n", len(written), len(scaffold.AuditProfiles()))
}
