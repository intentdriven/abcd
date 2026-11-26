---
schema_version: 1
id: "iss-2611260929594647"
slug: "after-a-preflight-fails-nothing-points-a"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "landing the night branches while away, 2026-11-26"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "When make preflight fails, end its output with the narrow rerun for every failing test, grouped by package and in the stage's own mode (go test -race -count=3 -run '<names>' <pkg> for a race-stage failure, the plain form for the unit stage, the audit's own command for a gate), beside make precheck; and add the rule to the LOAD or a preflight rules domain: fix, run that rerun and make precheck clean, then rerun the full preflight once. Test that a failing fixture run prints the rerun lines."
---

After make preflight fails, nothing points a session at a narrow rerun of what failed before the next full run, so a fix is proven only by another hour-long preflight, and a second failure hidden behind the first costs another hour. On 2026-11-26 a landing preflight ran 2h10m and failed on two tests in one file, a data race caught only by the race stage and a refusal-shape audit; the fix for the first alone, rerun in full, would have failed again on the second. The preflight's failure output names the failing tests but not the command that reruns them in the mode they failed in.
