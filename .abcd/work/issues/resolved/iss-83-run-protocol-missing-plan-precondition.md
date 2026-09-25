---
schema_version: 1
id: "iss-83"
slug: "run-protocol-missing-plan-precondition"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "2026-07-12 /abcd:run drain #2 loop launch"
found_at: ".abcd/development/plans/2026-07-12-abcd-run-protocol.md"
resolution: "the shipped itd-94 gives the deterministic step-0 readiness gate (exit 1 is a skip, never a guess) that replaces the /abcd:run plan precondition"
impact: internal
resolved_by:
  intent: "itd-94"
---

The /abcd:run protocol has no precondition that the PLAN file itself exists: its fail-closed check covers a missing or malformed mandatory PLAN *field* (ROLE / RECOVER STATE) but not a wholly-absent PLAN *file*. In the drain #2 run the /loop fired against a PLAN_PATH that did not resolve on HEAD because the plan was being authored concurrently in another worktree and only merged mid-session (HEAD moved 54fef5f -> 137f8de under the running session). The runner correctly fail-closed but the gap invited synthesizing a substitute plan. Fix (protocol robustness): add an explicit step 0 — resolve PLAN_PATH on current HEAD; if the file does not exist, STOP and report without synthesizing a replacement plan (synthesis re-introduces the guess-the-gate risk fail-closed exists to prevent). Add an operational note: a plan-driven loop must only be launched after its PLAN is merged to the branch it reads. Secondary: at burst start git fetch + re-derive HEAD rather than trusting the session-start snapshot — here both the snapshot and NEXT.md were stale (NEXT.md described already-committed itd-81 work as uncommitted). Acceptance: the protocol names a missing-plan-file STOP distinct from missing-field, and forbids substitute-plan synthesis.

## Grounds

- pursued: a run whose plan does not resolve stops instead of synthesising one; shown wrong if a run proceeds without a resolvable plan
