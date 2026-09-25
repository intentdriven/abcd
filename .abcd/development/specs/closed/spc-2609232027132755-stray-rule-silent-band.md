---
id: spc-2609232027132755
slug: stray-rule-silent-band
intent: itd-2609231434459890
origin: researcher-authored
production_mode: hand-written
---
# stray-rule-silent-band

## Summary

spc-2609231542463113 delivered the load check as specified, but its stray rule does not deliver what itd-2609231434459890's Mechanism, Decision 1 and Grounds claim of it. This spec holds that remainder, and the intent stays in `planned/` until it is settled.

The remainder is the stray rule's silent band. A process is a stray only when its lifetime share of a core is at least 0.9, and the extreme trigger fires only when the one-minute load is strictly above four times the online cores. Busy loops that have run for days, of any account, each hold their fair share of the machine, cores divided by their count. On 16 cores at the default limits, 17 loops (0.94 of a core each) warn through the stray trigger; 18 to 64 loops, a load between 1.125 and 4 times the cores, each hold under 0.9 of a core and raise no trigger at all: no warning and no run-log event; from 65 loops the warning is the extreme trigger's alone. The band is pinned by `TestStrayRuleSilentBand` in `internal/core/machineload/classify_test.go`, and the defect is iss-2609231947544298.

What settles it is the product thinker's ruling H1 of 2026-09-25 on the stray definition, recorded in `.abcd/work/DECISIONS.md`: busy for its share. A long-running process outside abcd's lanes is a stray when it uses nearly all the CPU it could get on the machine as loaded, its share measured against its fair share (the cores divided by the runnable demand), not against a fixed 0.9 of a core; own and other accounts' processes are judged alike, other accounts' stay counted only, and it is neither a second sample nor a summed-cores trigger.

## Approach

`machineload.FairShare` is the CPU one runnable process can get on the machine as the snapshot finds it loaded: the online cores divided by the one-minute load average, the runnable demand, capped at one core, and one core when there is no load reading or no core count. `Classify` makes a process older than the stray limit a stray when its lifetime share is at least `NearFullShare` (0.9) of that fair share, before the uid split. On a machine loaded no higher than its cores the rule is the near-full core it was; above that the threshold falls with the load, so n busy loops on 16 cores at load n, each holding 16/n of a core, are strays at every n. The extreme trigger is unchanged. The one snapshot decides.

The warning names what a program can get when that is under a core ("Load 40.0 on 16 online cores: a program can get about 40% of a core", or "At that load ..." after the extreme line), computed from the load and core count a `load` event already carries, so a logged event still renders exactly what was printed. The CLI help, the generated CLI reference, `commands/implement.md` and the brief's load-check section describe the fair-share rule. The check still warns and never refuses.

## Tests

- `TestStrayRuleCoversTheOversubscribedBand` (successor to `TestStrayRuleSilentBand`): 17, 18, 20, 40, 64, 65 and 440 two-day loops at load n on 16 cores warn through the stray trigger, with the extreme trigger from 65, and every loop is counted, for another account's loops and the caller's own alike.
- `TestStrayShareIsRelativeToTheFairShare` and `TestFairShare`: the boundary at 0.9 of the fair share under load, the full-core rule at or under the cores, and one core with no load reading.
- `TestMechanismFlagsIncidentTwo`: the 440 incident's 38 loops are counted beside the extreme trigger.
- `TestEightConcurrentPreflightsAreQuiet` holds: at load 41.91 the machine's long-lived programs (0.19 of a core at most) stay under 0.9 of the 0.38 a program can get.
- `TestImplementLoadWarnsInTheOversubscribedBand`: the rendered warning names the caller's loops, counts the other account's and states what a program can get.
