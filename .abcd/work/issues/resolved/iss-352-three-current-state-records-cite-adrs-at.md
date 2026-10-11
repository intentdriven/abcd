---
schema_version: 1
id: "iss-352"
slug: "three-current-state-records-cite-adrs-at"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/development/brief/03-evidence/03-open-questions.md"
resolution: "three citations repointed at decisions/adrs/ as markdown links the blocker rule gates"
impact: internal
---

three current-state records cite ADRs at development/research/adr/, a path that does not exist and a layout adr-30 explicitly rejected; all three are backticked code spans links_resolve never checks
## Evidence

- `brief/03-evidence/03-open-questions.md:28` (live routing list; the two sibling pointers resolve), `brief/03-evidence/04-tradeoffs.md:69` (meta-section about the file's present role — line 3 of the same file links `../../decisions/adrs/` correctly), `intents/disciplines/itd-37-modification-grammar.md:75` (active discipline, present tense). `development/research/adr/` does not exist; all ADRs live at `development/decisions/adrs/`, and adr-30:66 names `research/adr/` as a rejected alternative (that site is intended narration — untouched).
- All three are backticked code spans, invisible to `links_resolve` (`internal/core/lint/lint.go:1398` matches markdown links only) — a distinct gap from iss-303 (fragments).
- The 2026-07-06 plan-consistency review caught this at four sites; one was fixed, three remain, no ledger handle carried the remainder.
- Refuter verdict: CONFIRMED substantive. Fix: repoint all three; render as markdown links so the class is gated.
