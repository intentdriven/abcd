---
schema_version: 1
id: "iss-2611101316559868"
slug: "abcd-build-has-no-mode-that-runs-its"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Add a dry run to build that runs every start check (key, readiness, eligibility, peers, footprint overlap) and prints the rows it would act on, reserving, writing and pushing nothing; a test runs it on a buildable record and finds no reservation and no run made."
---

abcd build has no mode that runs its start checks without reserving or starting anything, so finding out what build would decide about a record takes a real build, which reserves it, or a throwaway program. On 9 Nov at 02:50 a lane wrote a throwaway program to prove the peers check against the real checkout, because a real build would have reserved the intent.
