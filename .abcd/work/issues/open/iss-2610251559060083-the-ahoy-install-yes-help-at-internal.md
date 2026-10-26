---
schema_version: 1
id: "iss-2610251559060083"
slug: "the-ahoy-install-yes-help-at-internal"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Reword the --yes help to name all six exclusions: the pin, the status line and the routing tables need an answered prompt; the README question, the drain rule and the conventions-file removal are asked only of a person at a terminal (--readme-parts answers the README question anywhere). Keep backquotes out, then regenerate the page."
---

The ahoy install --yes help at internal/surface/cli/cli.go:4261 (docs/reference/cli/commands.md:135) says --yes excludes 'the optional git-identity pin, the status line and the model-tier routing tables, which need an answered prompt (run without --yes, or answer every prompt with: yes | abcd ahoy install)'. Under --yes the install also leaves the README question, the drain eligibility record and the removal of a conventions file that only repeats AGENTS.md: optionalGapIDs (internal/core/ahoy/apply.go:2359) lists them, and optionalSkipped returns every pending one under Yes (apply.go:2378-2381). Those three are asked only at a terminal (terminalOnlyGapIDs, apply.go:2366), and piped stdin is not a terminal (stdinPrompter.AtTerminal, cli.go:4766), so the advised yes | abcd ahoy install never answers them. The install's own result already says so (optionalSkipReason, cli.go:4597-4614). The drain-rule (a101c59d1) and conventions-file (c97f6807b) offers joined the list after the help was worded (db74fb520), and so did the README parts (13615ee16).
