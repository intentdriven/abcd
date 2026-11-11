---
schema_version: 1
id: "iss-2611110928038046"
slug: "go-toolchain-1-26-10-fails-govulncheck"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "CI govulncheck job on every branch, 2026-11-11"
origin: researcher-authored
production_mode: hand-written
found_at: "go.mod"
remedy: "Bump go.mod's go directive to 1.26.11 (scripts/pinned-toolchain.sh and every setup-go step read go.mod), move the runbook's contained test image tag with it, and confirm govulncheck v1.7.0 reports no reachable vulnerability on that toolchain"
resolution: "go.mod moved to go 1.26.11; every setup-go step and scripts/pinned-toolchain.sh read the version from go.mod, and the runbook's contained test image moved to golang:1.26.11, which TestTheIntakeRunbookImageTracksGoMod holds to go.mod; govulncheck v1.7.0 reports no reachable vulnerability on that toolchain"
impact: fix
---

Go toolchain 1.26.10 fails govulncheck on GO-2026-6859, a public Go standard-library advisory in net/http, found in go1.26.10 and fixed in go1.26.11. govulncheck v1.7.0 reports it as reached by the module's code, so CI's govulncheck job failed on every branch. The external-contribution runbook's contained test image (.abcd/work/intake.md) tracks go.mod and moves with it.
