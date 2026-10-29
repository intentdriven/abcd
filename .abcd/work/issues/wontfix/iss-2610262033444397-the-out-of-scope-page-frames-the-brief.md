---
schema_version: 1
id: "iss-2610262033444397"
slug: "the-out-of-scope-page-frames-the-brief"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 434 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md:3"
remedy: "none (filed automatically)"
finding_ref: cfk-f2b96b1b8605f602
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040267270. Line 3 ('eight planned phases') and line 172 ('scoped into a phase') are the same stale premise in one page, against the same adr-2609212115255771 retirement."
duplicates: [iss-2610262040267270]
---

The out-of-scope page frames the brief as the work of eight planned phases; the reflect chapter says adr-2609212115255771 retired the phase.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/03-out-of-scope.md:3` — "This brief describes the work bundled into the eight planned phases"
- End B: `.abcd/development/brief/04-surfaces/09-reflect.md:13` — "2609212115255771-phases-and-milestones-are-retired.md) retired the phase; itd-24 decisions 4 and 5)"

03-out-of-scope opens by saying the brief describes the work bundled into the eight planned phases, and later has an intent enter work by being scoped into a phase. 09-reflect says adr-2609212115255771 retired the phase, which is why a retrospective's grain is the release. If the phase is retired there are no planned phases for the brief's work to be bundled into, so one of the two premises is stale.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 434.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040267270. Line 3 ('eight planned phases') and line 172 ('scoped into a phase') are the same stale premise in one page, against the same adr-2609212115255771 retirement.
