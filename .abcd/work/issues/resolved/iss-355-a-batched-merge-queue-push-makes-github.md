---
schema_version: 1
id: "iss-355"
slug: "a-batched-merge-queue-push-makes-github"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".github/workflows/release.yml"
resolution: "the release content commit is derived from the receipts directory instead of HEAD^2^ ancestry, so a batched merge-queue tip cannot misresolve it"
impact: fix
resolved_by:
  commit: "9106d2b5"
---

a batched merge-queue push makes github.sha the batch tip, so auto-release tags the wrong commit and release.yml's HEAD-caret-2-caret receipt derivation resolves an unrelated PR's commit whenever the release roll is not the last entry in its queue batch; the ancestry guard passes and the run wedges after the immutable tag, the iss-326 outcome by a route itd-93's pre-merge check cannot close because batch composition is decided by the queue
## Evidence

- Merge queues here merge one commit per PR but advance main in batched pushes: the 2026-08-19 14:29-14:31 block landed five merge commits with auto-release push runs only for two of them (runs 202-204 by consecutive run_number), so `github.sha` on a push event is the batch tip. `auto-release.yml:139` tags `github.sha`; `release.yml:225-229` derives receipts from `HEAD^2^` of the tagged commit; `release.yml:233` (`git merge-base --is-ancestor`) asserts ancestry only, which an unrelated batch-mate satisfies. A release roll merged as a non-final batch entry therefore tags the wrong commit and wedges exactly as iss-326 records for the missing-receipts cause — but by a route the itd-93 pre-merge receipts check cannot close, because batch composition is decided by the queue at merge time from whatever else is green (`.abcd/work/rulesets/main-protection.json`, `max_entries_to_merge: 5`).
- The rehearsal (`release.yml:420-441`) hardcodes the two-commit no-ff shape, so it can never surface this. `main-review.json`'s squash/rebase merge methods are NOT an independent path (the queue's MERGE method governs).
- Refuter verdict: CONFIRMED substantive (major). Recorded, not fixed: the fix belongs in a required-check workflow this environment cannot validate without a real Actions run (the iss-301/302 precedent). Proposed fix: (a) derive the content commit by locating the `.abcd/work/reviews/<sha>/` receipts directory (walk `HEAD^2^`, `HEAD^`, then first-parent merges' `^2^` bounded), aborting loudly before any tag when none matches, mirrored into the rehearsal plus a batched-queue simulation; or (b) set `max_entries_to_merge: 1` (keep `max_entries_to_build: 5`), restoring the one-merge-per-push invariant the derivation assumes.
