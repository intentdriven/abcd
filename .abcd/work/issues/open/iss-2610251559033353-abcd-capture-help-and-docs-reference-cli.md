---
schema_version: 1
id: "iss-2610251559033353"
slug: "abcd-capture-help-and-docs-reference-cli"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Drop the backquotes from the --remedy usage string at internal/surface/cli/cli.go:5322, add a test that no flag usage string in the tree carries a backquote, and regenerate the reference page."
---

abcd capture --help and docs/reference/cli/commands.md:492 render the remedy flag as --remedy abcd capture remedy the proposed fix, one line (required); abcd capture remedy rewrites it later. The usage string at internal/surface/cli/cli.go:5322 backquotes abcd capture remedy. pflag's UnquoteUsage takes the first backquoted span as the flag's value placeholder, so the placeholder reads as a command instead of string. This is the class iss-2608301212428844 resolved for --grounds. TestGroundsFlagUsageRendersAStringPlaceholder (internal/surface/cli/capture_surface_test.go:1220) pins only the three --grounds flags, so --remedy slipped through. It is the only such placeholder on the page.
