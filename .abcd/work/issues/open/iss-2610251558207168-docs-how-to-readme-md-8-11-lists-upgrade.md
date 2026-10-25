---
schema_version: 1
id: "iss-2610251558207168"
slug: "docs-how-to-readme-md-8-11-lists-upgrade"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D2"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/README.md"
remedy: "Write docs/how-to/upgrade-to-v0.15.0.md listing each break with its replacement and add it to the index, or record a decision that a breaking release's changelog line is its only guide."
---

docs/how-to/README.md:8-11 lists upgrade guides for v0.12.0, v0.13.0 and v0.14.0 only. v0.15.0 is also a breaking release (CHANGELOG.md:82-84): bare abcd decide records a decision and an ADR needs --adr, and ahoy install's --outbound-hook and --outbound-forge are refused at exit 2 in favour of --outbound (internal/surface/cli/ahoy_retired_flags.go:11-14). A script written for v0.14 has no guide.
