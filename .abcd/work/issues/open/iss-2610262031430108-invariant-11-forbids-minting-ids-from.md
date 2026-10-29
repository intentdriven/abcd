---
schema_version: 1
id: "iss-2610262031430108"
slug: "invariant-11-forbids-minting-ids-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 169 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:159"
remedy: "In itd-44's routing section and operator-deviation block, replace `adr-<next-free>-<slug>.md` and 'next-free numbering' with the id `abcd decide --adr` mints (timestamp-numeric, adr-45), which the draft's own delivery-state note already names."
related_intents: [itd-44]
finding_ref: cfk-12496318663d7183
---

Invariant 11 forbids minting ids from the current maximum, but itd-44 has capture write `adr-<next-free>-<slug>.md`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:159` — "the mint never allocates by looking at the current maximum (two minters sharing a stale view must still produce distinct ids)"
- End B: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:122` — "`adr-<next-free>-<slug>.md` into `.abcd/development/decisions/adrs/`"

Invariant 11 says the mint never allocates by looking at the current maximum, so two minters sharing a stale view still produce distinct ids (adr-45, timestamp-numeric ids). itd-44 routes a confirmed `decision` verdict to an ADR named `adr-<next-free>-<slug>.md`. That is next-free allocation, the scheme the invariant rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 169.
