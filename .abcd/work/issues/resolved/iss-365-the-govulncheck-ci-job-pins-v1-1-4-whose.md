---
schema_version: 1
id: "iss-365"
slug: "the-govulncheck-ci-job-pins-v1-1-4-whose"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".github/workflows/ci.yml"
resolution: "Bumped the govulncheck CI pin from @v1.1.4 to @v1.7.0; verified locally that v1.7.0 loads go1.25 packages (reaches the DB-fetch stage) where v1.1.4 failed at package loading."
impact: fix
---

The govulncheck CI job pins @v1.1.4 whose bundled x/tools caps its type-checker at go1.24 while go.mod declares go 1.25.6, so package loading fails and the job exits before any scan; non-gating masks it as advisory, so no vuln analysis has run since it landed
## Evidence

- `.github/workflows/ci.yml:379-393` — job `govulncheck` runs `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` under setup-go `1.25.13`; no `-C`, no `|| true`, no `continue-on-error`; non-gating (absent from `.abcd/work/rulesets/main-protection.json` required contexts).
- `go.mod:3` — `go 1.25.6`.

Reproduced locally (go1.25.6): `@v1.1.4` fails at `loading packages … package requires newer Go version go1.25 (application built with go1.24)` for every package and exits 1 before any vuln-DB fetch. Bisected the fix: `@v1.2.0` is the first version that loads go1.25; `@v1.7.0` (latest) loads cleanly and reaches the DB-fetch stage.

## Adversarial verdict

CONFIRMED (substantive). Distinct from the round-2 refutation (which established non-gating is *deliberate*; this is the orthogonal observation that the scan never *runs*). In-scope: a one-token CI tool-pin bump, no go.mod dependency, locally verifiable through package loading. Fix: bump `@v1.1.4`→`@v1.7.0` on ci.yml:393.
