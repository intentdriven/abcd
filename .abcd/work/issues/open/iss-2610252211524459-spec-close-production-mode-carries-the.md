---
schema_version: 1
id: "iss-2610252211524459"
slug: "spec-close-production-mode-carries-the"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Give spec close its own --production-mode help, e.g. 'how the --remainder spec's text was produced: ...; only with --remainder'."
---

spec close --production-mode carries the shared productionModeFlagHelp, 'how this record's text was produced ... (default: the repo's declared mode, else hand-written)' (internal/surface/cli/cli.go:3350, wired at cli.go:3930, docs/reference/cli/commands.md:4773). On spec close the flag only stamps the follow-on spec that --remainder mints, and without --remainder it is refused with exit 2 (cli.go:3813-3815). The flag help says neither.
