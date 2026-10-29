---
schema_version: 1
id: "iss-2610262032080136"
slug: "the-press-release-lists-the-itd-29-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 226 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:27"
remedy: "none (filed automatically)"
related_intents: [itd-2609201925079472]
finding_ref: cfk-dee49336ac0a003f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301816. Same press release phrase, with the shipped pace intent as the other end recording that the hand verbs were dropped with itd-29."
duplicates: [iss-2610262030301816]
---

The press release lists the itd-29 `/abcd:run` operator surface as in-scope plumbing; itd-2609201925079472 says those hand verbs were dropped with itd-29.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:27` — "plus operator-internal command wiring (e.g. `/abcd:run`, the itd-29 autonomous-run operator surface"
- End B: `.abcd/development/intents/shipped/itd-2609201925079472-an-autonomous-implementation-run-paces.md:45` — "Mid-run telemetry and an operator's hand verbs over the state file: Dropped with `itd-29` until wanted."

The brief's in-scope plumbing includes `/abcd:run`, the itd-29 surface with status, pause, resume and preflight. The shipped pace intent says the operator's hand verbs over the state file were dropped with itd-29 until wanted, and that itd-29 was superseded. The press release claims scope the record has dropped.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 226.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301816. Same press release phrase, with the shipped pace intent as the other end recording that the hand verbs were dropped with itd-29.
