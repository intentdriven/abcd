---
schema_version: 1
id: "iss-2610262032125902"
slug: "itd-42-s-worked-example-says-invariant-3"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 235 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:34"
remedy: "none (filed automatically)"
related_intents: [itd-42]
finding_ref: cfk-f4766f40b6ac7fd1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Invariant 3 is the native default front door, so the example mis-anchors, but it sits in the re-scoped press release and the shipped pre-pass anchors conflicts to real invariant numbers."
---

itd-42's worked example says invariant 3 is about config never being written outside `~/.abcd/`; the invariants register's invariant 3 is the native default front door

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:34` — "Invariant 3 says config is never written outside `~/.abcd/` — your intent implies it is; which gives?"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:13` — "**Native default front door** — abcd's logic lives in a transport-agnostic Go core"

The itd-42 press release gives, as its example of a grounded conflict the grill raises, 'Invariant 3 says config is never written outside `~/.abcd/`'. The invariants register numbers invariant 3 as 'Native default front door', which is about abcd's logic living in a transport-agnostic Go core and has nothing to do with where config is written. The example ties a numbered invariant to content that invariant does not hold, so it teaches the wrong anchor.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 235.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Invariant 3 is the native default front door, so the example mis-anchors, but it sits in the re-scoped press release and the shipped pre-pass anchors conflicts to real invariant numbers.
