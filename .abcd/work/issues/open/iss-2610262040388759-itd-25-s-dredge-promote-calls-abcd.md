---
schema_version: 1
id: "iss-2610262040388759"
slug: "itd-25-s-dredge-promote-calls-abcd"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 945 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-b9049345774e2ee4
---

itd-25's dredge promote calls /abcd:intent new; the skills chapter says the canonical create is /abcd:intent with quoted text.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36` — "calls `/abcd:intent new` with the cluster summary"
- End B: `.abcd/development/brief/05-internals/08-skills.md:127` — "it extends the canonical create `/abcd:intent "<text>"`"

itd-25 has dredge promote fold a cluster into a draft by calling /abcd:intent new. 08-skills names the canonical create as /abcd:intent "< text>", bare quoted text with no 'new' sub-verb. That gives two names for one act, and one of them names a verb that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 945.
