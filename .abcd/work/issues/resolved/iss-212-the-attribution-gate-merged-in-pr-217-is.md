---
schema_version: 1
id: "iss-212"
slug: "the-attribution-gate-merged-in-pr-217-is"
severity: "major"
category: "process"
source: "user-observation"
found_during: "post-merge verification of PR #217 (2026-08-11)"
found_at: ".github/workflows/attribution.yml"
resolution: "attribution added to the main-protection ruleset's required status checks via ruleset update; the dependabot precondition (skipped conclusion does not block) was settled empirically 2026-08-11, so the wiring completes the gate"
impact: internal
---

The attribution gate merged in PR #217 is dead scaffolding until it is added to the branch's required status checks, and it is not there. Verified 2026-08-11 after the merge: main's required contexts are still check (macos-latest), check (ubuntu-latest), gitleaks, record-lint, smoke, zizmor — no "attribution" — and enforce_admins is false. The workflow runs and reports green on every pull request, so the repo now LOOKS gated while nothing is actually blocked; that is the false green the loud-staging principle refuses, and it fails the "wired or it isn't done" boundary the same way (no dead scaffolding). The gate was built precisely because the maintainer said it must be certain they do not have to check for attribution drift by hand; in its current state that certainty does not exist, and the appearance of it is worse than the previous honest absence.

There is a PRECONDITION that must be settled before the required-check wiring, or fixing this breaks something else. .github/workflows/attribution.yml carries a job-level condition excluding bot authors, and GitHub blocks a pull request on a required check that never reports ("Expected — waiting for status"). If a job skipped by that condition does not emit a check run with conclusion "skipped", making attribution required turns every dependabot PR permanently unmergeable — the exact outcome the bot exemption exists to avoid. This was NOT settled empirically: PR #211 was the only bot PR available and its branch predated the attribution workflow, so no attribution check ran on it and the question stayed open.

Two ways out. Verify on the next dependabot PR whose branch contains the workflow that a check run named "attribution" appears with conclusion "skipped"; or remove the ambiguity by dropping the job-level condition and exempting bots INSIDE the step (exit 0 early), so the job always runs, always reports success, and no skipped-check semantics are relied on. The second is preferred and is roughly a five-line change — the script already exempts bot commits, so only the body half needs it.

Also outstanding and deliberately left to the maintainer: enforce_admins is false, so once attribution is required it binds every contributor and agent except the one person who merges everything.
**Precondition SETTLED empirically 2026-08-11 — the skipped-check worry does
not hold.** This record states the question stayed open because "PR #211 was
the only bot PR available and its branch predated the attribution workflow, so
no attribution check ran on it." That ceased to be true while #211 was being
repaired: clearing its BEHIND state merged main — and therefore
`.github/workflows/attribution.yml` — onto the dependabot branch, so the
workflow WAS present on head `98c8152` when CI ran.

The result, read from the API rather than the PR page: `git cat-file -e
98c8152:.github/workflows/attribution.yml` succeeds, `gh pr view 211 --json
author -q .author.is_bot` is `true`, and the head's check runs include
`name=attribution status=completed conclusion=skipped` (check-run id
93735741608). So a job excluded by the job-level `if:` DOES emit a check run,
with conclusion `skipped`, on a genuine bot pull request.

That removes the blocker this record placed in front of the required-check
wiring: the feared outcome was a required check that "never reports", leaving
every dependabot PR permanently unmergeable, and the check demonstrably
reports. Adding `attribution` to the required contexts no longer needs the
five-line restructure to be settled first.

Two things this evidence does NOT settle, left deliberately. Whether branch
protection treats a `skipped` conclusion as satisfying a required context is
GitHub-side behaviour this observation does not exercise — it proves the check
appears, not how protection scores it. And `enforce_admins` remains false, so
the maintainer note at the end of this record stands untouched.

