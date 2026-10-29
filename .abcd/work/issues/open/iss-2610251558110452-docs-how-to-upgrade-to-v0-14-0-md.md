---
schema_version: 1
id: "iss-2610251558110452"
slug: "docs-how-to-upgrade-to-v0-14-0-md"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D2"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/upgrade-to-v0.14.0.md"
remedy: "Say the lists are v0.14.0's and point at abcd --help and abcd --help --agent for the current set, or bring both lists up to date."
---

docs/how-to/upgrade-to-v0.14.0.md:103-108 gives the /abcd: menu and the agents-only pages as v0.14.0 had them, in the present tense. Since then the menu also lists embargo (v0.14.2), land (v0.15.0) and status (v0.15.1), and the agents-only pages include memory and reserve (v0.14.2) and announce (v0.15.1) (internal/surface/cli/helpgroups.go:120-155; commands/*.md block and user-invocable frontmatter).
