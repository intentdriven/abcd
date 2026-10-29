---
schema_version: 1
id: "iss-2610262040545166"
slug: "the-glossary-says-abcd-never-owns-the"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 963 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-2609201916151817]
finding_ref: cfk-fff9fa952ca41fc3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037018200. Same End A. itd-2609201916151817 states 'refines adr-27: this is the thin native loop it names, with its pieces', so the intent and the ADR agree and the glossary line is the outlier."
duplicates: [iss-2610262037018200]
---

The glossary says abcd never owns the run loop; the build intent makes abcd's own verb that loop.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop."
- End B: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:60` — "The orchestrator is the verb's own loop over a state file, never a model."

loop.md defines the run loop as the receipt-gated iteration over ready work and states flatly that abcd never owns it, a pluggable adapter supplying it. itd-2609201916151817 makes `abcd build`/`abcd implement` exactly that loop, run by abcd's own binary over a state file, verifying each lane's receipt before advancing. One says the receipt-gated loop is always an adapter's, the other builds it into the verb, so a builder cannot follow both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 963.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037018200. Same End A. itd-2609201916151817 states 'refines adr-27: this is the thin native loop it names, with its pieces', so the intent and the ADR agree and the glossary line is the outlier.
