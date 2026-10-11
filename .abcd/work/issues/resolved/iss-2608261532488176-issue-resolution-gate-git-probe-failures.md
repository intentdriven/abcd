---
schema_version: 1
id: "iss-2608261532488176"
slug: "issue-resolution-gate-git-probe-failures"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "scripts/check-issue-resolution.sh"
resolution: "All four git probes on the gate rc-check and exit 2 with stderr surfaced; empty-ledger pass stays loud; cases pin both fault shapes and the clean pass"
impact: internal
resolved_by:
  commit: "b3d4c18c"
---

The issue-resolution gate reports OK having scanned zero records whenever a git probe fails. scripts/check-issue-resolution.sh ledger arm: cd "$(git rev-parse --show-toplevel)" collapses to cd '' (rc 0) when the substitution fails under set -e; the ls-tree listing carries || true so a git failure reads as an empty ledger and check_ledger returns 0 with 'no ledger records — nothing to check'; and the is-shallow probe compares a failed substitution against 'true', so the exit-2 environment-fault arm is disarmed by the same fault it exists to report. Reproduced: GIT_TEST_ASSUME_DIFFERENT_OWNER=1 (git's dubious-ownership refusal, the reachable local form for containers/sudo/devcontainers) turns 113-records-checked into OK exit 0; from a non-git cwd likewise. The commits arm fails closed (bare assignment) — divergence within one file, and RS003 is the sole detector for resolved_by.commit shas rewritten by the squash/rebase merges the repo permits. Same class as the round-9 check-reviews.sh fail-closed rewrite (every git probe rc-checked, exit 2), which left this sibling untouched. Detector: cases asserting exit 2 under a failing git; acceptance: every git probe on the ledger arm is rc-checked and a git fault exits 2, with the legitimate empty-ledger pass kept loud.