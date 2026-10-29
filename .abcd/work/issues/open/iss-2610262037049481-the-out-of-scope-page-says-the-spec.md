---
schema_version: 1
id: "iss-2610262037049481"
slug: "the-out-of-scope-page-says-the-spec"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 695 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md:172"
remedy: "Say the close is run by abcd build's landing stage in the lane's worktree, or by hand in the change that lands the work; drop 'nothing promotes a record on its own'."
finding_ref: cfk-510efab1af9b5dab
---

The out-of-scope page says the spec close that ships an intent is a manual step; the build loop's landing runs the close itself.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/03-out-of-scope.md:172` — "the close is a manual step run in the change that lands the work"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:743` — "In the lane's worktree it runs the spec's close (which ships the intent and parks the fidelity receipt)"

03-out-of-scope.md says nothing promotes a record on its own and the close is a manual step run in the change that lands the work. 34-build.md says the landing stage runs the spec's close in the lane's worktree, which ships the intent with no person running it. A build run does promote a record on its own, so the manual-step claim does not hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 695.
