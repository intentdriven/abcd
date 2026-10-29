---
schema_version: 1
id: "iss-2610262032046139"
slug: "the-register-reserves-voyage-for-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 217 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:231"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-0623ca298e0417d1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-36 used voyage in the sense adr-35 later retired, and shipped intents are not rewritten; the glossary names the open lifecycle-term question itself."
---

The register reserves `voyage` for the operator-level operations namespace; itd-36 uses `voyage` to mean a project abcd manages.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:231` — "The **operations** namespace (verb-side: what we did), as against `lifeboat` (noun-side: what gets carried)"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:23` — "a per-`voyage` curated knowledge substrate that compounds across sessions"

The reserved-vocabulary table defines `voyage/` as the operations namespace (what was done in an embark or disembark), as against `lifeboat`. itd-36's press release calls the memory store a per-`voyage` substrate, so the word means a managed project there. One name stands for two concepts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 217.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-36 used voyage in the sense adr-35 later retired, and shipped intents are not rewritten; the glossary names the open lifecycle-term question itself.
