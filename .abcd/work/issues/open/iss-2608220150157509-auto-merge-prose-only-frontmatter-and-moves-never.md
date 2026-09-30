---
schema_version: 1
id: "iss-2608220150157509"
slug: "auto-merge-prose-only-frontmatter-and-moves-never"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcdev-site decision interview 2026-08-22"
found_at: ".github (auto-merge convention)"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Plan a mechanical CI rung refusing auto-merge arming on frontmatter or record-path diffs?"
remedy: "Waits on planning ruling H (plan the mechanical rung): if planned, add a workflow on the pull_request auto_merge_enabled event that runs a scripts/ check over the diff and, when it changes YAML frontmatter or adds, moves or deletes a file under a record store, disables auto-merge and comments why, proven by a cases script in the style of scripts/check-issue-resolution-cases.sh with a prose-only and a frontmatter fixture; if not planned, keep the protocol and resolve this record as wontfix with that reason."
---

Auto-merge for docs PRs is armable only when the diff is prose-only, defined precisely: no YAML frontmatter changes AND no file adds, moves, or deletes under the record stores — because record state lives in both frontmatter (status flips) and directory position (lifecycle bucket moves). A state change is a gate crossing (adversarial-review-scales-with-blast-radius) and always waits for the human; prose fixes are below the review floor. Adopted as documented protocol now (applied by eye when arming); the mechanical rung — a CI check that refuses auto-merge arming when the diff touches frontmatter or record paths — is a later seed per script-first-mvp

## Remedy grounds (2026-09-29)

- GitHub Actions fires pull_request with the auto_merge_enabled activity type (https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows, consulted 2026-09-29) and auto-merge proceeds once required reviews and checks pass (https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request, consulted 2026-09-29), so disarming at the moment of arming refuses only the automatic merge and leaves the human merge open.
- Rejected: a ruleset 'Restrict file paths' rule, which acts on pushes and would block every legitimate record change, and a required failing check, which would block the human merge too.
