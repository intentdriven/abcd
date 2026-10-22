---
schema_version: 1
id: "iss-2610222018028530"
slug: "go-toolchain-1-26-9-fails-govulncheck-on"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "pull request 1107 govulncheck job, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "go.mod"
remedy: "Bump go.mod's go directive to 1.26.10 (scripts/pinned-toolchain.sh and every setup-go step read go.mod), run make preflight on the new toolchain, and confirm govulncheck v1.7.0 reports no reachable vulnerability"
resolution: "go.mod moved to go 1.26.10; govulncheck v1.7.0 reports no reachable vulnerability on that toolchain, and every setup-go step and scripts/pinned-toolchain.sh read the version from go.mod; the runbook's contained test image moved to golang:1.26.10 with it, and TestTheIntakeRunbookImageTracksGoMod holds that tag to go.mod. The product thinker signed off this toolchain move for the 2026-10-23 release on 2026-10-22, asked directly by the coordinating session"
impact: fix
---

Go toolchain 1.26.9 fails govulncheck on GO-2026-6809 (crypto/x509: distrust compromised intermediate certificates), a public Go standard-library advisory fixed in go1.26.10. govulncheck v1.7.0 reports it on every pull request's govulncheck job. Found on pull request 1107's govulncheck job, 2026-10-22. The external-contribution runbook's contained test image (.abcd/work/intake.md) also named go 1.26.7, two bumps behind go.mod, and moves with it.

## Grounds

- pursued: a binary built from this tree carries the crypto/x509 fixed in go1.26.10; govulncheck reporting GO-2026-6809 on the next cut's binary would show it wrong
