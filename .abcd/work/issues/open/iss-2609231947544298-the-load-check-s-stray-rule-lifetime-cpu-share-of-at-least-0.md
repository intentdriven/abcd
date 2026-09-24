---
schema_version: 1
id: "iss-2609231947544298"
slug: "the-load-check-s-stray-rule-lifetime-cpu-share-of-at-least-0"
severity: "major"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/classify.go"
deferred_after: "v0.9.0"
deferral_reason: "Settling the band needs the product thinker's ruling on the stray definition (a share relative to what the machine could give the process, a second sample, or an amended claim), and the product thinker is away; itd-2609231434459890 is held in planned/ with this band as the remainder spec spc-2609232027132755 until they rule, so the over-claim does not ship."
---

The load check's stray rule is silent across a load band that itd-2609231434459890's Mechanism, Decision 1 and Grounds claim it covers. A process is a stray only when it is older than the stray limit and its lifetime share of a core is at least 0.9 (NearFullShare, internal/core/machineload/classify.go); the extreme trigger fires only when the one-minute load is strictly above four times the online cores. Busy loops that have run for days each hold their fair share, cores divided by their count, so on 16 cores at the default limits: 17 loops (0.94 of a core each) warn through the stray trigger; 18, 20, 40 or 64 loops, a load between 1.125 and 4 times the cores, each hold under 0.9 of a core and raise no trigger, so the check is silent (no warning, no run-log event); from 65 loops the warning is the extreme trigger's alone, as at the 2026-09-22/23 incident's load of about 440, where the stray count read 0. The share filter applies before the uid split, so the band silences the caller's own loops as well as another account's. The fixture that shows it is TestStrayRuleSilentBand in internal/core/machineload/classify_test.go (TestMechanismFlagsIncidentTwo pins the 440 case). Major: the check's main purpose, flagging a forgotten busy loop, fails silently in the band, and the intent's Grounds say it is shown wrong by exactly that. The remainder spec spc-2609232027132755 holds the intent planned on this band. Fixing it needs a ruling on the stray definition (a share relative to what the machine could give the process, or a second sample), which the lane cannot take.
