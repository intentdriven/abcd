---
schema_version: 1
id: "iss-2609012313465609"
slug: "every-pull-request-pays-two-full-ci"
severity: "major"
category: "process"
source: "user-observation"
found_during: "pr-queue-observation-2026-09-02"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Waits on the M19 planning interview (which direction first): if queue batching, raise min_entries_to_merge_wait_minutes above 0 in .abcd/work/rulesets/main-protection.json and apply it; if the full matrix in the queue only, run the macOS, race and smoke jobs on the merge_group event alone in .github/workflows/ci.yml; if caching, profile the macOS job first, since setup-go caches by default; if hooksPath, have ahoy install set core.hooksPath as an owned change. Prove the pick by the CI cycles per landed pull request, measured before and after."
deferred_after: v0.11.1
deferral_reason: "The product thinker's ruling M19 of 2026-09-23: planned next cycle as its own intent that chooses among the record's directions, not folded into itd-115. Since filing, the pre-push hook checks a preflight receipt instead of running the preflight (2026-09-25) and the macOS leg's cap rose to 45 minutes (ruling Z, 2026-09-28); neither removes the second CI cycle. Owed: that intent, which opens on one question: which direction first, hooksPath at install, the full matrix in the queue only, caching, or queue batching?"
---

Measured on 2026-09-01 with thirteen auto-merge pull requests in flight: each one pays two full CI cycles before it lands, and the macOS check job alone takes about 13 minutes (ubuntu 9). Cycle one: the pull request is armed, main moves, the strict up-to-date policy makes it BEHIND, the keep-current script updates the branch, and the full CI re-runs on the updated head before the pull request is CLEAN enough to enter the merge queue. Cycle two: the merge queue runs the full CI again on the merge group. Every merge moves main and knocks the not-yet-queued pull requests back to cycle one, so a batch of thirteen cost roughly twenty-six 13-minute cycles serialised in ALLGREEN groups, and a one-line record change waited an hour. The maintainer asks how to speed the gates up, for example by running them locally first. Directions, none adopted: (1) local-first is already built and not wired on every account: make preflight is the pre-push gate and .githooks/pre-push runs it, but core.hooksPath is unset on at least one active account, so nothing runs before a push; wire it at ahoy install (an owned ConfigChange) and record which accounts have it; note that a local pass shortens nothing on the forge, it only stops red pushes. (2) Run the full matrix once, in the queue: on the pull_request event run the fast lane only (format, record gates, ubuntu build and test) and keep the macOS leg, the race lane and the smoke harness for the merge_group event, which already runs everything; the CI classifier that stands macOS down for docs-only changes shows the seam exists. (3) Cache the Go build and test cache across runs (actions/setup-go cache keyed on go.sum) and check whether the macOS job's 13 minutes is test time or cold-build time. (4) Let the queue batch: min_entries_to_merge_wait_minutes is 0, so each pull request tends to get its own group; a short wait lets several share one CI run. (5) The strict policy is the multiplier and was kept on 2026-09-01 (iss-2609012202237613); revisit only with the duplicate-id gate argument answered.

## Deferral 2026-09-29

Deferred past v0.11.1: The product thinker's ruling M19 of 2026-09-23: planned next cycle as its own intent that chooses among the record's directions, not folded into itd-115. Since filing, the pre-push hook checks a preflight receipt instead of running the preflight (2026-09-25) and the macOS leg's cap rose to 45 minutes (ruling Z, 2026-09-28); neither removes the second CI cycle. Owed: that intent, which opens on one question: which direction first, hooksPath at install, the full matrix in the queue only, caching, or queue batching?

## Remedy grounds (2026-09-29)

- Each direction is stated so the interview can pick one without further research; the measure is the record's own (two full cycles per pull request).
- SOTA check: GitHub's merge-queue documentation (https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue, read 2026-09-29) gives the minimum group size and its wait time for lengthy CI, and says the queue provides the benefits of the up-to-date requirement without the author updating the branch, which bears on direction 5 once the duplicate-id argument is answered; actions/setup-go (https://github.com/actions/setup-go, read 2026-09-29) enables caching by default, so direction 3 is largely in place.
- Rejected: dropping the strict policy now, which the record defers to the duplicate-id argument.

## Evidence 2026-10-07

The macOS leg hit its 45-minute job cap in the merge queue again. On #839's merge-group run (37590582904), `check (macos-latest)` ran 07:57:44 to 08:43:14 and was cancelled in `Test (race, internal)`, which had started at 08:14:47. The ubuntu leg was green. The queue dropped the pull request and it had to be re-queued by hand, adding a full cycle. This is the second time in two release cycles (the v0.13.1 release PR, 2026-10-05, was cancelled the same way). The queue retries nothing on a runner timeout, so an unattended run has to watch for it.

## Evidence 2026-10-08

The macOS leg's 45-minute cap recurred three more times in the overnight drain, though iss-2609281514435020 was resolved for the same class. PR #852's own macOS check failed at 45m31s (2026-10-07 23:11Z). PR #854 was dropped from the merge queue at 2026-10-08 00:36Z on a failed merge-group check, its own checks green. PR #867 was dropped at 07:43Z: its merge-group ci run 37740516314 had check (macos-latest) cancelled at the cap (07:07 to 07:52). Each needed a hand re-run or re-enqueue (iss-2610090642376032 covers the silent drop).
