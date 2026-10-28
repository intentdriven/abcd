---
schema_version: 1
id: "iss-2610262044170732"
slug: "the-design-fiction-draft-adds-a-format"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1196 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-30, itd-2609212130136102]
finding_ref: cfk-646b79de779e109e
---

The design-fiction draft adds a format flag to intent new, which the verb consolidation removed

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "`/abcd:intent new --format=fiction` opens a fiction-shaped interview"
- End B: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:40` — "`version --check` becomes `update --check`; `intent new` is removed."

itd-30's capture lane is `/abcd:intent new --format=fiction` and a fiction-shaped interview inside `/abcd:intent new`. itd-2609212130136102 removes `intent new` outright. The draft's main surface does not exist, so the draft as written cannot be built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1196.
