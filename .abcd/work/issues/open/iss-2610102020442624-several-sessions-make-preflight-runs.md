---
schema_version: 1
id: "iss-2610102020442624"
slug: "several-sessions-make-preflight-runs"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/implement.md"
remedy: "Give make preflight a machine-wide slot (a lock under the abcd home keyed on the root commit, or a ceiling of N concurrent preflights) that a run waits on and reports, so sessions' preflights queue instead of overlapping; give its test and race stages a go test -p derived from the online cores and the current load, since one -race run already starts about one test binary per core and two saturate the machine (load 199 on 16 cores at 01:46 on 2026-10-11 with three race runs of internal/surface/cli live); keep the warn-only extreme-load check for load the slot cannot see."
---

Several sessions' make preflight runs overlap on one machine with nothing to space them out, and the overlap fails gates on tests no lane touched. On 2026-10-10 evening, two other sessions' full test suites were running at once (two go-build trees, a dozen .test binaries at 40-80% CPU each) and the one-minute load reached 124 on 16 cores. Four drain lanes in a row then lost a preflight run to load-only timeouts: TestAnIssueLaneLandsOnePullRequestThatResolvesItsIssue and TestAClosedPullRequestIsRefusedAndNothingIsCleanedUp (internal/core/implement/loop), and two internal/core/interview tests ('WaitDelay expired'). Each passed alone. The preflight load check (implement load --site preflight) warns above four times the core count but never waits, so every lane pays a re-run of about 20 minutes, and a lane whose re-run also lands in a peak has its receipt refused.
