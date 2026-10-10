---
schema_version: 1
id: "iss-234"
slug: "docs-only-prs-pay-the-full-two-os-build"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "Inert-path skip gate landed (PR #244) and the rehearsal PR #246 auto-merged through it: classifier logged inert=true, smoke and zizmor reported skipped and satisfied the ruleset, macOS leg passed in 8s with heavy steps stood down, ubuntu unit lane ran in full"
impact: internal
---

docs-only PRs pay the full two-OS build+test matrix — add a fail-closed inert-path skip gate
## Evidence (2026-08-16, measured on PR #239 — a record-only change)

check (macos) 2m31s + check (ubuntu) 1m57s + smoke 20s + zizmor 10s +
gitleaks 6s + record-lint 4s, all ruleset-required. For a docs-only PR the
only strictly needed work is: gitleaks (docs can leak secrets), the
reviews-charter record-lint, attribution, and the check job's last two
steps — `go run ./cmd/record-lint` and `go run ./cmd/abcd docs lint` —
which compile most of the module anyway (the gates ARE the binary; a
zero-build docs lane is structurally impossible). Skippable on inert
changes: the full test suite, race tests, vet, the entire macOS leg,
smoke, and zizmor (unless `.github/**` changed).

## Design constraints (settled or recorded)

- **Fail-closed classifier only.** The 2026-08-15 decision records that a
  "no source touched" test FAILS OPEN here: `commands/*.md`,
  `hooks/hooks.json`, `.abcd/*.json`, and the rules router are markdown/
  JSON that change behaviour. Skip only when EVERY changed file matches a
  narrow inert allowlist (`docs/**`, `.abcd/development/**`,
  `.abcd/work/**`, root markdown); anything else runs the full matrix.
- **All seven check contexts are ruleset-required**, so workflow-level
  `paths:` filtering wedges PRs (never-reporting checks block). The viable
  shape is a `changes` classifier job + job-level `if:` on downstream
  jobs; skipped jobs report a `skipped` conclusion, documented as
  satisfying required checks — MUST be verified on a rehearsal PR before
  auto-merge relies on it (loud-staging: no assumed green).
- `ci.yml` is not scaffold-coupled (parity test pins release workflows
  only). Typed link: a proven skip gate `refines` what itd-106 (abcd
  scaffolds required CI per repo) would ship to managed repos.
- Repo is public — Actions minutes are free; the win is wall-clock /
  auto-merge latency (~2.5-3 min → ~1 min) and queue pressure, not money.
- CI config changes need explicit maintainer sign-off before landing.

Related: iss-233 (duplicate push+PR runs) is the cheaper, independent
first fix — do it regardless of this one.
