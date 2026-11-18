---
schema_version: 1
id: "iss-2611040554426316"
slug: "the-land-chapter-says-mergegate-replaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/38-land.md"
remedy: "Point the citation at dcn-2609300000000025 instead of 34-build.md."
resolution: "The land chapter's merge gate section now cites dcn-2609300000000025, the decision record holding ruling AM1, instead of 34-build.md, which never names it; a landing test pins the citation and that the record names AM1."
impact: internal
resolved_by:
  commit: "0eb0313a87405d224e829337aa7f3c0be5bc52ea"
---

The land chapter says mergeGate replaces ruling AM1, citing 34-build.md (.abcd/development/brief/04-surfaces/38-land.md:154). 34-build.md does not name AM1, and this line is the only occurrence under .abcd/development/brief, so the citation leads a reader to nothing. The ruling is recorded in .abcd/work/decisions/dcn-2609300000000025-the-implement-loop-arms-auto-merge-only.md.

## Grounds

- pursued: a reader following the citation for AM1 reaches the record that states it; shown wrong if dcn-2609300000000025 did not name AM1 or the chapter cited another home for the ruling.
