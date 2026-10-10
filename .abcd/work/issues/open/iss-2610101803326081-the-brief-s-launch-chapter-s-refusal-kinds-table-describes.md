---
schema_version: 1
id: "iss-2610101803326081"
slug: "the-brief-s-launch-chapter-s-refusal-kinds-table-describes"
severity: "nitpick"
category: "drift"
source: "review-followup"
found_during: "abcd-60 drain run 2026-10-10, lane for iss-2610072347238476"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Reword the unfixed-finding row of the refusal-kinds table in .abcd/development/brief/04-surfaces/04-launch.md to name both cases (a consequential finding open since the anchor, and an open issue record that differs from HEAD), once the verb split's launch step has landed or with its owner's word."
---

The brief's launch chapter's refusal-kinds table describes unfixed-finding as the consequential (major/critical) finding case only, while the guardrail list forty lines below now says it also fires for an open issue record that differs from HEAD (added by the drain lane for iss-2610072347238476, which left the table row alone because a peer's verb split edits nearby). The table and the list now disagree within one chapter.
