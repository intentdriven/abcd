---
schema_version: 1
id: "iss-2610251559107915"
slug: "abcd-build-next-help-internal-surface"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Add the footprint exclusion to build next's candidate list (with a remote named origin, an intent whose spec names no footprint packages is excluded at the check footprint) and confine the zero-score sentence to a checkout without origin, then regenerate the page."
---

abcd build next --help (internal/surface/cli/build.go:303-310; docs/reference/cli/commands.md:417-425) defines the candidates as the planned intents that pass every abcd build <itd-N> check, 'less one this checkout already has a run in progress for'. It also says a part whose section is absent reads zero, with the reason saying the spec carries no footprint. In a checkout with a remote named origin, candidatesAt excludes, before any scoring, an intent whose spec's ## Footprint names no packages, at the check footprint (internal/core/implement/loop/next.go:149-186), so such an intent is never scored. Commit 6eb4ac9c0 added the exclusion and documented it in commands/build.md:193 and the brief, but left the CLI help unchanged.
