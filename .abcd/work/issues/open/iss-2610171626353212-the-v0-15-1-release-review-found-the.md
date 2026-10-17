---
schema_version: 1
id: "iss-2610171626353212"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/39-status.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/39-status.md:130 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/39-status.md:130 out of step with the shipped code. It says: Exits: exit 0 when the fetch succeeded or no origin; exit 1 when the fetch failed; exit 2 outside a git checkout or with an operand. In the code: internal/surface/cli/status.go also exits 1 when what was fetched could not be read (ReadStatus error) and in a checkout without its local tier (v.NoTier), and `abcd status --help` says so ('exit 1 when it failed, when what it fetched could not be read, or in a checkout without its local tier'). The brief omits both e…
