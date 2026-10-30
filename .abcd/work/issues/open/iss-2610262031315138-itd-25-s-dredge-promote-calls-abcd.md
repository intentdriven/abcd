---
schema_version: 1
id: "iss-2610262031315138"
slug: "itd-25-s-dredge-promote-calls-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 141 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36"
remedy: "In itd-25 (dredge promote and its acceptance criterion) replace `/abcd:intent new` with the quoted create, `/abcd:intent` with the cluster summary as its quoted text; `new` was removed and the verb refuses it. N:1 links need their own writer, since `capture promote` is strictly 1:1."
related_intents: [itd-25]
finding_ref: cfk-27d524a9cfe02871
---

itd-25's dredge promote calls /abcd:intent new, which the brief says is refused

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36` — "(calls `/abcd:intent new` with the cluster summary"
- End B: `.abcd/development/brief/01-product/02-context.md:11` — "`new` is not a sub-verb: `abcd intent new` is refused and names the quoted create."

itd-25 has dredge promote fold a cluster into a new intent draft by calling /abcd:intent new with the cluster summary. The context chapter says new is not a sub-verb and that abcd intent new is refused, naming the quoted create. The promotion step depends on a verb the intent surface does not offer.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 141.
