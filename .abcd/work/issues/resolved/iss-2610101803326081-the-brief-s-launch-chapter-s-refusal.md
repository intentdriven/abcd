---
schema_version: 1
id: "iss-2610101803326081"
slug: "the-brief-s-launch-chapter-s-refusal"
severity: "nitpick"
category: "drift"
source: "review-followup"
found_during: "abcd-60 drain run 2026-10-10, lane for iss-2610072347238476"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Reword the unfixed-finding row of the refusal-kinds table in .abcd/development/brief/04-surfaces/04-launch.md to name both cases (a consequential finding open since the anchor, and an open issue record that differs from HEAD), once the verb split's launch step has landed or with its owner's word."
resolution: "The launch brief's refusal-kinds row for unfixed-finding now names both cases Emit raises under it: a consequential finding this cycle captured that is still open and unanswered, and an open issue record that differs from HEAD at any severity, matching the guardrail list in the same chapter; the kind's doc comment in emit.go says the same, and a test holds the row to both halves."
impact: internal
resolved_by:
  commit: "58c15b44c1dd447604d6fe594b57a8598e4162b8"
---

The brief's launch chapter's refusal-kinds table describes unfixed-finding as the consequential (major/critical) finding case only, while the guardrail list forty lines below now says it also fires for an open issue record that differs from HEAD (added by the drain lane for iss-2610072347238476, which left the table row alone because a peer's verb split edits nearby). The table and the list now disagree within one chapter.

## Grounds

- pursued: the table row, the guardrail list, commands/launch.md and Emit's three RefusalUnfixedFinding appends all describe the same two cases; a row that again names only one case, or Emit raising unfixed-finding for a case the row does not name, would show it wrong, and TestBriefRefusalKindsRowForUnfixedFindingNamesBothCases fails on the first.
