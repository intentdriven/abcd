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

What settles it is the product thinker's ruling on the stray definition (for example, a share relative to what the machine could give the process, or a second sample), or an amendment of the intent's claim. Neither is taken here: the intent's promise is the product thinker's, and they are away.
