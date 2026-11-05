---
id: spc-2611050505250705
slug: the-restart-reads-the-forge-and-the
intent: itd-2611041549461012
origin: researcher-authored
production_mode: hand-written
---
# the-restart-reads-the-forge-and-the

## Summary

**What the restart path still owes after spc-2611041706004822.** The record, the listing and the take-over have landed: a successor lists every run from what it wrote down, collects before it restarts, and starts nothing twice. What remains reads what the listing does not yet: a pull request's state on the forge and the records a crashed copy stranded, and each agent's own process from the launch ledger in place of the process that started it. iss-2610110915210571, which the closed spec's step 2 named, was fixed by d7c05e9cb before this branch, with its test; this work added the stop's clearing of a judgement and the listing that names one, and the record is resolved naming that commit. The intent stays planned until this spec closes.

## Footprint

- packages: internal/core/implement/loop/resume.go
- tests: internal/core/implement/loop/resume_test.go

## Steps

1. **Open pull requests a crash left drifting** (AFTER)
   - priority: AFTER
   - login: none
   - needs: none
   - packages: internal/core/implement/loop/resume.go
   - tests: internal/core/implement/loop/resume_test.go
   The listing names each pull request from the state without a network read; reading the forge for its queue state (iss-2610241414414823) and sweeping records a crashed copy stranded (iss-2610241107435264) are later.
2. **Each agent's own process, from the launch ledger** (AFTER)
   - priority: AFTER
   - login: none
   - needs: none
   - packages: internal/core/implement/loop/resume.go
   - tests: internal/core/implement/loop/resume_test.go
   Once itd-2611031342011261 lands, an agent is alive or lost by the process its ledger entry names rather than by the process that started it.
