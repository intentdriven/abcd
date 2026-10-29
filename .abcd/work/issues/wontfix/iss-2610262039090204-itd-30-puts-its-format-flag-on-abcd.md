---
schema_version: 1
id: "iss-2610262039090204"
slug: "itd-30-puts-its-format-flag-on-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 843 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-30]
finding_ref: cfk-5ffd2164bf71b9b3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262044170732. Same draft edit as iss-2610262044170732, set against a different statement of the same current state."
duplicates: [iss-2610262044170732]
---

itd-30 puts its format flag on `/abcd:intent new`, but universal patterns says the canonical create is bare `/abcd:intent "<text>"`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "\`/abcd:intent new --format=fiction"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:74` — "the canonical create `/abcd:intent "<text>"` only writes a draft"

itd-30 adds `--format=fiction` to a `new` sub-verb of `/abcd:intent`, with a `refine` sub-verb beside it. 04-universal-patterns says the canonical create is `/abcd:intent "<text>"`, which writes a draft. The same act has two invocation names, so as itd-30 spells it, the flag has no shipped verb to attach to.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 843.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262044170732. Same draft edit as iss-2610262044170732, set against a different statement of the same current state.
