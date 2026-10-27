---
schema_version: 1
id: "iss-2610262035222673"
slug: "the-lint-chapter-calls-the-outbound-gate"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 567 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:91"
remedy: "none (filed automatically)"
finding_ref: cfk-32d25be7f7be7a6c
---

The lint chapter calls the outbound gate the only surface that judges harness-leak text before it is public, but the guard enforces a forge-text check on gh pr and gh issue text before it is posted

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:91` — "This is the fourth, and it is the only one that judges text BEFORE it is"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:102` — "(a posting `gh pr` or `gh issue` command whose title or body carries a live"

16-lint counts four surfaces that judge the session-URL and tool-footer class and calls `abcd lint outbound` the fourth and the only one that judges text before it is public. 17-guard enforces `forge-text-outbound`, which refuses a posting gh pr or gh issue command whose title or body carries a live agent-session URL or a tool attribution footer. That check also judges the class before the text is public, so the count and the 'only one' claim cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 567.
