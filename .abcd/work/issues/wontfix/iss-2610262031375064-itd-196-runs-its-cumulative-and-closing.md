---
schema_version: 1
id: "iss-2610262031375064"
slug: "itd-196-runs-its-cumulative-and-closing"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 153 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-196]
finding_ref: cfk-3700454b6db51cfc
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042291241. Same unit problem in itd-196 as iss-2610262042291241, cited against the mental model."
duplicates: [iss-2610262042291241]
---

itd-196 runs its cumulative and closing rehearsals per phase, while the mental model retires the phase

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:40` — "The cumulative rung is what turns a phase into something more than a list of"
- End B: `.abcd/development/brief/01-product/03-mental-model.md:56` — "The phase and the milestone are retired; the documents under"

itd-196's second and third rungs run what the phase can deliver so far and everything the phase set out to deliver at the phase's close, and say the cumulative rung turns a phase into something more than a list of merged intents. The mental model says the phase and the milestone are retired, with no planned end condition beyond each intent's criteria and the derived release. Two of the discipline's three rungs are keyed to a unit the record no longer has, so nothing triggers them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 153.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042291241. Same unit problem in itd-196 as iss-2610262042291241, cited against the mental model.
