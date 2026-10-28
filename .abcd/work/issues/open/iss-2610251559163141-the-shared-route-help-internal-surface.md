---
schema_version: 1
id: "iss-2610251559163141"
slug: "the-shared-route-help-internal-surface"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/route.go"
remedy: "Word the --route help so it holds for a dual-mode verb (a verb run without its payload flag dispatches none and refuses --route), then regenerate the page."
---

The shared --route help (internal/surface/cli/route.go:86-91) says 'each invocation dispatches one', and it renders for disembark principles, press-release and review (docs/reference/cli/commands.md:1033, 1052, 1083). Run without its payload flag, each of those verbs runs its deterministic mode and dispatches no agent (delegatedAgent, route.go:73-78). It then refuses --route with exit 2 (route.go:115-120). Only graveyard, whose --lessons-json is required, always dispatches one.
