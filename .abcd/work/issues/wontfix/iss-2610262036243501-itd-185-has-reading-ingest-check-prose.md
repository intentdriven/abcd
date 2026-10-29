---
schema_version: 1
id: "iss-2610262036243501"
slug: "itd-185-has-reading-ingest-check-prose"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 645 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold.md:57"
remedy: "none (filed automatically)"
related_intents: [itd-185]
finding_ref: cfk-0d18ebfb5a2640c7
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. ingest_regime.go says the signature registry is gone rather than softened; the stale scope bullet is superseded inside the same intent by the ruling that names it."
---

itd-185 has reading ingest check prose that ranks, settles or proposes; the reading chapter says that prose-signature layer was removed and only fields are refused

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold.md:57` — "Semantic violations — prose that ranks, settles, or proposes without the fields — are checked too"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:262` — "It is gone rather than softened, because a gate that reads prose is still reading prose."

itd-185's scope ships semantic checks over an item's prose beside the structural regime table, to degrade only on observed noise. 23-reading says the regime gate refuses only a real decision field, and that the second layer, a registry of named signatures over an item's prose, is gone outright. Ingest cannot both read item prose for ranking, settling or proposing and not read it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 645.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. ingest_regime.go says the signature registry is gone rather than softened; the stale scope bullet is superseded inside the same intent by the ruling that names it.
