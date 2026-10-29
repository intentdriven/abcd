---
schema_version: 1
id: "iss-2610262039262634"
slug: "the-build-intent-arms-auto-merge"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 860 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:62"
remedy: "none (filed automatically)"
related_intents: [itd-2609201916151817]
finding_ref: cfk-f36ea247f6e1b3d0
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037294601. Same AM1 contradiction checked against the land page instead of the build chapter."
duplicates: [iss-2610262037294601]
---

The build intent arms auto-merge wherever the ruleset gates merges through a queue, but the land page says AM1 arms it only where a queue and a required approval both exist

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:62` — "A pull request with auto-merge armed where the ruleset gates it through a queue, a pull request left open where it does not."
- End B: `.abcd/development/brief/04-surfaces/38-land.md:125` — "auto-merge is armed only where a merge queue gates the default branch and a"

Decision 3 of itd-2609201916151817 arms auto-merge on a pull request wherever the ruleset gates it through a queue. The land page says the shared merge gate follows ruling AM1, which arms auto-merge only where a merge queue gates the default branch and a ruleset requires a person's approval. A repository with a queue but no required approval is armed under the intent and left alone under the land page.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 860.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037294601. Same AM1 contradiction checked against the land page instead of the build chapter.
