---
schema_version: 1
id: "iss-2610262044201056"
slug: "the-audit-draft-adds-a-separate"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1199 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-85, itd-2609212130136102]
finding_ref: cfk-846935fc91807da4
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034151413. Same draft edit as iss-2610262034151413, set against a different statement of the same current state."
duplicates: [iss-2610262034151413]
---

The audit draft adds a separate repo-conformance check verb, while the shipped consolidation folded every 'check this repository' spelling into one lint

A scope leakage (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:26` — "A new read-only `abcd audit` verb, distinct from `ahoy doctor` (doctor = tool-setup health; audit = repo conformance)."
- End B: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:21` — "its five spellings of "check this repository", so the person's list holds about a dozen verbs"

itd-85 creates `abcd audit` as a distinct verb for checking whether a repository conforms, beside an `ahoy doctor` sub-verb. itd-2609212130136102 removes the five spellings of 'check this repository' in favour of `abcd lint` with targets, and turns ahoy's modes into flags. Repo conformance checking is claimed by both lint and the proposed audit verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1199.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034151413. Same draft edit as iss-2610262034151413, set against a different statement of the same current state.
