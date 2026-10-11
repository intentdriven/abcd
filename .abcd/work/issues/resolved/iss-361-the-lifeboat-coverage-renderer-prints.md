---
schema_version: 1
id: "iss-361"
slug: "the-lifeboat-coverage-renderer-prints"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/core/lifeboat/coverage.go"
resolution: "Routed the per-repo and aggregate coverage renders' repo-derived strings (tiers, section name, status, tier, confidence, repo name) through the existing sanitize/termsafe; column widths computed from sanitised strings. Test TestRenderSanitisesUntrustedCoverageFields watched fail then pass."
impact: fix
---

The lifeboat coverage renderer prints untrusted cross-repo status and tiers_present fields to the terminal without termsafe, so a crafted probe report injects raw ESC/OSC-8/C1/bidi sequences; only the repo name is sanitised
## Evidence

- `internal/core/lifeboat/coverage.go:212` sanitises only `c.Repo.Name`; `:224` copies `TiersPresent` verbatim; `:258-265` `statusOf` returns `Status` verbatim.
- `internal/core/lifeboat/mapping.go:30,50` — `Tier`/`Status` are bare `string`; `Status.Valid()` (`:77`) is called only in tests, never on the decode/render path.
- `internal/surface/cli/cli.go:532-535` documents the operand as untrusted cross-repo; `:553-565` validates only `schema_version`; `:574-575` renders via `agg.Render()` to stdout.
- Aggregate render sinks: `coverage.go:273-276` (tiers line) and `:301-305` (`row.Cells`); sibling per-repo render shares the gap at `:98,:104,:106`.

Reproduced: a crafted `status`/`tiers_present` injects raw `ESC[31m` (SGR) and OSC-8 hyperlink to the terminal; repo name correctly defanged in the same run.

## Adversarial verdict

CONFIRMED (substantive). Refuter reproduced raw ESC/OSC-8 reaching the terminal. Distinct site from iss-259/iss-264/iss-340. Fix: route the tier/status/section strings through the existing `sanitize` at both the aggregate and per-repo render, with a pinning test.
