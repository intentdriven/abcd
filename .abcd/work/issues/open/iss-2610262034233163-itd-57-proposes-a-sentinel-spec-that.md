---
schema_version: 1
id: "iss-2610262034233163"
slug: "itd-57-proposes-a-sentinel-spec-that"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 492 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17"
remedy: "Supersede itd-57 (abcd intent reclassify naming the resolution of iss-2609200830076665 in the reason; record a one-line decision first if --by needs an intent or ADR) or re-scope it: a person-lifted hold already ships as `abcd intent hold` and `unhold`, a held line that build's start checks, the pick and the board's head refuse, so the sentinel-spec design is not needed."
related_intents: [itd-57]
finding_ref: cfk-a37b178d672987dc
---

itd-57 proposes a sentinel spec that keeps planned work out of autonomous pickup; the board chapter says the build's start checks already skip a held record.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17` — "that any future spec can depend on to stay invisible to autonomous pickup until a human explicitly unblocks it"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:344` — "the claim sections answered, no hold, no unsettled blocker, a step left to"

itd-57 promises a way to park planned work: a sentinel spec dependency that keeps a spec out of autonomous pickup until a person lifts it. 08-abcd.md says the build's pre-start checks, which both the pick and the board's head run, require no hold, so a held record is already skipped. Two mechanisms claim the same person-lifted block on autonomous pickup.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 492.
