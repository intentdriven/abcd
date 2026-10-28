---
schema_version: 1
id: "iss-2610262043297244"
slug: "itd-25-s-dredge-promote-calls-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1141 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-25, itd-2609212130136102]
finding_ref: cfk-520d166fa5743f3b
---

itd-25's dredge promote calls /abcd:intent new, which itd-2609212130136102 removed

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36` — "calls `/abcd:intent new` with the cluster summary"
- End B: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:40` — "`version --check` becomes `update --check`; `intent new` is removed"

itd-25 defines dredge promote as calling /abcd:intent new with the cluster summary. The shipped verb consolidation removes intent new outright. The draft depends on a verb the shipped record deleted, so its promotion path has nothing to call as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1141.
