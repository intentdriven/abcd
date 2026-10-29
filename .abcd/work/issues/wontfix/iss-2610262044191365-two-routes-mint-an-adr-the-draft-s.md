---
schema_version: 1
id: "iss-2610262044191365"
slug: "two-routes-mint-an-adr-the-draft-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1198 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:121"
remedy: "none (filed automatically)"
related_intents: [itd-44, itd-2609151138388536]
finding_ref: cfk-04eb6dc42332006f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034191519. Same draft edit as iss-2610262034191519, set against a different statement of the same current state."
duplicates: [iss-2610262034191519]
---

Two routes mint an ADR: the draft's capture writes adr-< next-free>, and the shipped decide verb mints ADRs with --adr

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:121` — "capture writes `adr-<next-free>-<slug>.md` into `.abcd/development/decisions/adrs/`"
- End B: `.abcd/development/intents/shipped/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md:185` — "and `abcd decide --adr "<title>"` mints an ADR"

itd-44 has capture write a sequentially numbered `adr-<next-free>-<slug>.md` into the ADR store. itd-2609151138388536 makes `abcd decide --adr` the verb that mints an ADR through the shared timestamp id seam. ADR minting is claimed twice, and the two use incompatible id schemes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1198.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034191519. Same draft edit as iss-2610262034191519, set against a different statement of the same current state.
