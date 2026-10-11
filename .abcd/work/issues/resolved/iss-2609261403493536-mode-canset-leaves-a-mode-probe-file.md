---
schema_version: 1
id: "iss-2609261403493536"
slug: "mode-canset-leaves-a-mode-probe-file"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mode/question.go"
resolution: "CanSet sweeps every regular file named exactly as a probe before it probes, best-effort, and treats its own probe already gone (a concurrent sweep took it) as success. TestCanSetSweepsTheProbesAnEarlierCallLeftBehind was watched red against the unswept tier; TestCanSetConcurrentProbesAllAnswer guards the concurrent case."
impact: internal
resolved_by:
  commit: "18fa82d8"
---

mode.CanSet leaves a .mode-probe-* file behind forever when the local tier turns unwritable between the probe's create and its remove (a chmod during the hook's window, or a kill between the two): CanSet reports 'removing the probe' once, the file stays, and nothing ever sweeps it. Review2 of the badge lane reproduced it: 40 probes against a chmod-flapping tier left 7 probe files. Remedy: sweep stale probe files at the top of CanSet.

## Grounds

- pursued: after a probe is orphaned, the next CanSet leaves the tier holding no probe-shaped regular file while a lookalike file and a probe-named directory survive, and concurrent probes all answer nil; a probe file surviving a later CanSet, or a concurrent probe reporting 'removing the probe', would show it wrong
