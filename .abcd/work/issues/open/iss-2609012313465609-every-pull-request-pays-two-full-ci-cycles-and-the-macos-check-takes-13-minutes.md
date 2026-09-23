---
schema_version: 1
id: "iss-2609012313465609"
slug: "every-pull-request-pays-two-full-ci-cycles-and-the-macos-check-takes-13-minutes"
severity: "major"
category: "process"
source: "user-observation"
found_during: "pr-queue-observation-2026-09-02"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M19: plan next cycle as its own intent choosing among the recorded directions (hooksPath at install, queue-only full matrix, caching, queue batching), not folded into itd-115). Earlier deferral: Cost, not correctness, and the remedy touches the path this release merges through. The finding is that a pull request pays two full CI cycles with a thirteen-minute macOS leg; nothing it describes produces a wrong answer, and the change that would fix it edits the same concurrency and merge-queue behaviour that iss-2609020716560392 is repairing in this cut. Sequencing a cost optimisation behind a correctness fix in the same machinery is the reason to wait, not a shortage of appetite."
---

Measured on 2026-09-01 with thirteen auto-merge pull requests in flight: each one pays two full CI cycles before it lands, and the macOS check job alone takes about 13 minutes (ubuntu 9). Cycle one: the pull request is armed, main moves, the strict up-to-date policy makes it BEHIND, the keep-current script updates the branch, and the full CI re-runs on the updated head before the pull request is CLEAN enough to enter the merge queue. Cycle two: the merge queue runs the full CI again on the merge group. Every merge moves main and knocks the not-yet-queued pull requests back to cycle one, so a batch of thirteen cost roughly twenty-six 13-minute cycles serialised in ALLGREEN groups, and a one-line record change waited an hour. The maintainer asks how to speed the gates up, for example by running them locally first. Directions, none adopted: (1) local-first is already built and not wired on every account: make preflight is the pre-push gate and .githooks/pre-push runs it, but core.hooksPath is unset on at least one active account, so nothing runs before a push; wire it at ahoy install (an owned ConfigChange) and record which accounts have it; note that a local pass shortens nothing on the forge, it only stops red pushes. (2) Run the full matrix once, in the queue: on the pull_request event run the fast lane only (format, record gates, ubuntu build and test) and keep the macOS leg, the race lane and the smoke harness for the merge_group event, which already runs everything; the CI classifier that stands macOS down for docs-only changes shows the seam exists. (3) Cache the Go build and test cache across runs (actions/setup-go cache keyed on go.sum) and check whether the macOS job's 13 minutes is test time or cold-build time. (4) Let the queue batch: min_entries_to_merge_wait_minutes is 0, so each pull request tends to get its own group; a short wait lets several share one CI run. (5) The strict policy is the multiplier and was kept on 2026-09-01 (iss-2609012202237613); revisit only with the duplicate-id gate argument answered.
