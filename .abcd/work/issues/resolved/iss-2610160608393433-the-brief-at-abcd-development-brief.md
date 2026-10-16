---
schema_version: 1
id: "iss-2610160608393433"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Reword the stale-intent row at 04-launch.md:517 to: an intent in planned/ whose every realising spec has closed."
resolution: "04-launch.md's stale-intent row now says the refusal fires on a planned intent whose every realising spec has closed, none open, as release/emit.go staleIntents does"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/04-launch.md:517 says refusal kind stale-intent is raised when 'an intent in planned/ has a spec that has closed'. The shipped rule (internal/core/release/emit.go:102-103) raises it only when a planned intent has no open spec left, every realising spec closed, as 05-intent.md:509 states.
