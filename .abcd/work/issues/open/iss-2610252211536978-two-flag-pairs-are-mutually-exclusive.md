---
schema_version: 1
id: "iss-2610252211536978"
slug: "two-flag-pairs-are-mutually-exclusive"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/site.go"
remedy: "Add '(not with --version)' to --preview and '(not with --yes)' to --check, or the reverse, then regenerate the reference."
---

Two flag pairs are mutually exclusive, but neither flag's help says so. site build --preview and --version (internal/surface/cli/site.go:88; help at site.go:83-87; docs/reference/cli/commands.md:4732-4733). update --check and --yes (internal/surface/cli/update.go:156; help at update.go:154-155; commands.md:4898-4899). A reader of the reference learns of either pairing only from cobra's error.
