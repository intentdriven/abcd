---
schema_version: 1
id: "iss-2610262030301816"
slug: "the-press-release-lists-abcd-run-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 8 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:27"
remedy: "Drop '/abcd:run' and the itd-29 operator surface from the 'Plumbing that makes it possible' bullet (line 27): itd-29 is superseded by itd-2609201916151817 (planned) and no run verb or commands/run.md exists."
finding_ref: cfk-0fac43cb23fe6cbc
resolution: "the design brief now states what the binary does, in present tense"
impact: internal
---

The press release lists `/abcd:run` as in-scope operator wiring; the scope page says it is not built and its intent is superseded

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:27` — "plus operator-internal command wiring (e.g. `/abcd:run`, the itd-29 autonomous-run operator surface — read-mostly `status`/`pause`/`resume`/`preflight` over the pluggable autonomous-run seam; not part of the user-facing command set)"
- End B: `.abcd/development/brief/01-product/04-scope.md:19` — "is not built: itd-29 is superseded (by itd-2609201916151817, planned), and no `run` verb or `commands/run.md` is on any shipped surface."

The press release's 'What's In Scope' lists `/abcd:run`, the itd-29 operator surface with status/pause/resume/preflight, as operator-internal wiring. The scope page says `/abcd:run` is not built, itd-29 is superseded, and no `run` verb or command page is on any shipped surface. A superseded surface cannot also be part of the plumbing the press release delivers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 8.
