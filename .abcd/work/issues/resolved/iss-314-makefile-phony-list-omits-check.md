---
schema_version: 1
id: "iss-314"
slug: "makefile-phony-list-omits-check"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "Makefile"
resolution: "check-attribution, scaffold-sync and scaffold-sync-check added to .PHONY."
impact: internal
---

Makefile PHONY list omits check-attribution scaffold-sync and scaffold-sync-check so a same-named root file silently no-ops the target
## Evidence
`Makefile:13` — `.PHONY: build test vet clean preflight lint-reviews record-lint docs-lint smoke`. Omitted: `check-attribution` (`:43`), `scaffold-sync` (`:66`), `scaffold-sync-check` (`:69`). `go build ./cmd/scaffold-sync` from the repo root drops a `scaffold-sync` binary there, and `/scaffold-sync` is not in `.gitignore` — a plausible natural collision that makes `make scaffold-sync` a silent no-op exit 0. `record-lint` is already in `.PHONY` and shares the exact `cmd/`-package-collision shape, so this is an internal inconsistency.

## Adversarial verdict: CONFIRMED (nitpick)
Reproduced: touch the names, `make scaffold-sync` → "up to date", rc 0. Blast radius bounded (go-test parity gates the actual drift). Fix: add the three names to `.PHONY`. Zero risk, restores an invariant the file half-holds.
