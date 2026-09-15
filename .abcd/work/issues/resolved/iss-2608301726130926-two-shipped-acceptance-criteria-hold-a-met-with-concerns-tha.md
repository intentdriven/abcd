---
schema_version: 1
id: "iss-2608301726130926"
slug: "two-shipped-acceptance-criteria-hold-a-met-with-concerns-tha"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "phase-4-conditional"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped"
resolution: "itd-185 landed and merged as 1d84ce26, so the condition discharges on its first branch: the five MET_WITH_CONCERNS criteria across itd-178, itd-180 and itd-189 stand as issued and need no re-issuing. The producer they rested on exists and is wired, reached from both the CLI and the plugin markdown surface."
impact: additive
resolved_by:
  intent: "itd-185"
---

two shipped acceptance criteria hold a MET_WITH_CONCERNS that becomes NOT_MET if itd-185 does not land in this phase

The facilitator's verdict rule, now filed as itd-192, says a criterion whose
producer does not exist is `MET_WITH_CONCERNS` if THIS PHASE wires it, and
`NOT_MET` if not. Two verdicts were issued under the first branch:

- itd-178 `ac-2`
- itd-180 `ac-1`

Both are `MET_WITH_CONCERNS` **because spc-63 / itd-185 is in scope**. itd-185
is the ingest door, and as of 2026-08-30 it sits in `intents/planned/` and has
not been started. So the condition under which those two verdicts were issued
is not yet satisfied, and it is the phase, not the criterion, that decides it.

**If itd-185 does not land in this phase, both flip to `NOT_MET` and both
verdicts need re-issuing.** That is not a judgement call at the time; it is the
rule itd-192 states, applied to a fact the phase will have settled.

This record exists because the obligation had no marker. It lived in one line of
a handover file and in the verdict's own prose, both of which are read by
whoever happens to be reading them. A verdict that silently keeps a rating whose
stated precondition failed is exactly the shape the fidelity audit exists to
catch, so leaving the check to memory would be the audit failing in its own
terms.

**Trigger: Phase 4, before the pull request to `main`.** Resolve it one of two
ways and record which:

- itd-185 landed: both verdicts stand as issued, and this closes on that fact.
- itd-185 did not land: re-issue both verdicts as `NOT_MET`, with the reason a
  promised intent went unwired, and close this citing the re-issue.

Do not close it by deciding the criteria are fine. The rule does not leave that
open.

**WIDENED 2026-08-31 by the itd-189 fidelity audit.** This record was filed
naming two criteria. It is at least five, across three intents.

itd-189 shipped with all three of its criteria at MET_WITH_CONCERNS, and the
reason is the same one: `capture.IngestReading` mints a reading item and has no
caller outside its own package, so the admission and surprise schemas are armed
over a corpus the shipped product cannot populate
(iss-2608310912206941). Every one of its Given clauses presupposes an artefact
only itd-185 can produce.

So the Phase-4 check is now:

- itd-178 `ac-2` — MWC, flips to NOT_MET
- itd-180 `ac-1` — MWC, flips to NOT_MET
- itd-189 `ac-1`, `ac-2`, `ac-3` — MWC, all three rest on itd-185

If itd-185 lands, all five stand as issued. If it does not, five criteria across
three shipped intents need re-issuing as NOT_MET, and three of them are the
whole of one intent's acceptance.

That changes what this record is for. It was a reminder; it is now the single
place recording that one unstarted intent carries the acceptance of three
shipped ones.

## Grounds

- pursued: the record armed a two-way condition on whether itd-185 would land this phase, and it landed, so the five criteria resting on it hold as issued; what would show this wrong is the ingest verb failing to reach the reading-record family from a front door, which would leave the corpus unpopulated and the criteria unsupported after all.
