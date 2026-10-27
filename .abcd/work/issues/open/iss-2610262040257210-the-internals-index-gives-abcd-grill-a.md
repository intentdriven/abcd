---
schema_version: 1
id: "iss-2610262040257210"
slug: "the-internals-index-gives-abcd-grill-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 930 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md:14"
remedy: "none (filed automatically)"
finding_ref: cfk-c9c6870a12fe7eec
---

The internals index gives /abcd:grill a design target of promotion to /abcd:intent grill; the build sequence says the grill is superseded by itd-94's planning interview.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/README.md:14` — "its **design target** is promotion to `/abcd:intent grill`"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:101` — "under itd-94, which supersedes it"

The internals index says the grill's design target is promotion to /abcd:intent grill. The build sequence says no grill sub-verb exists because itd-94's planning interview supersedes the grilling itd-27 asked for. A superseded surface cannot also be a pending design target.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 930.
