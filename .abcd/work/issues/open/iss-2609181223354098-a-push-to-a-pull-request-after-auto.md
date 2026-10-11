---
schema_version: 1
id: "iss-2609181223354098"
slug: "a-push-to-a-pull-request-after-auto"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "Gropius managed-repo session gropiusllm-56, relayed to abcd-17 on 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules (bundled COMMITTING domain)"
remedy: "Waits on the product thinker's routing ruling: add the line 'after arming auto-merge on a merge queue, do not push again; to push, disarm, push, re-arm, and confirm the merged commit contains the last local commit before deleting the branch' to the bundled COMMITTING rules in internal/core/rules/defaults/rules.json if that home is chosen, to .abcd/work/intake.md if the runbook is, or to both. Prove the rules home with a rules test that the COMMITTING domain renders the line."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): Should the bundled COMMITTING domain carry the line: after arming auto-merge on a merge queue, do not push again; to push, disarm, push, re-arm, and confirm the merged sha contains the last local commit before deleting the branch? The record also offers the intake runbook as a second home."
---

A push to a pull request after auto-merge is armed on a merge queue can be dropped silently, and the bundled COMMITTING rules do not say so. Reported from the Gropius managed repository on 2026-09-18 (its own ledger carries it under its own id): with auto-merge armed and the entry already queued, a later push to the branch was not what the queue merged; the repository lost two commits and re-landed them in a second pull request. The failure is the forge's, not abcd's: a queue builds its entry from the head it took at enqueue time, and a push after that either removes the entry (to be re-armed) or, when the entry is already merging, lands nothing, and the pull request's merged state looks the same either way. The lesson is portable to every managed repository whose ruleset routes through a queue, which is why the session asked for a line in the bundled COMMITTING domain: after arming auto-merge on a queue, do not push again; if a push is needed, disarm first, push, re-arm, and confirm the merged sha contains the last local commit before deleting the branch. Sibling of iss-172 (queued entries left BEHIND) and iss-2609020716561070 (the up-to-date requirement beside a queue); this one is the dropped push, which neither covers. Recorded as a process observation; the routing (a COMMITTING rule, a line in the intake runbook, or both) is the product thinker's.

## Remedy grounds (2026-09-29)

- The line is portable to every managed repository with a queue, which argues for a bundled rule; the runbook reaches only this repository's intake.
- SOTA check: GitHub's auto-merge and merge-queue pages (https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request and https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/merging-a-pull-request-with-a-merge-queue, read 2026-09-29) do not say what a push to a queued entry does, so the rule cannot defer to the forge's documentation and the confirm-the-merged-commit step is the only safe check.
- Rejected: a guard refusing git push on an armed branch, which would need a forge query per push.
