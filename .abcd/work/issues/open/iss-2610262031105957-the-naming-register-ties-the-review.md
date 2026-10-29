---
schema_version: 1
id: "iss-2610262031105957"
slug: "the-naming-register-ties-the-review"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 94 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:181"
remedy: "In 04-naming.md line 181 change the review bucket's '(family-1 verdicts)' to say a lifeboat review returns family 1 while a change reviewer states its own closed words (SHIP/FIX FIRST, APPROVE/BLOCK/NEEDS-INPUT), as the 01-agents.md Verdict-tag protocol records."
finding_ref: cfk-6b8d360a3ec3eb30
---

The naming register ties the `review` bucket (a change judged) to family-1 verdicts, but the agent catalog says family 1 judges only lifeboats and the change reviewers' verdicts are not family 1

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:181` — "`review` compares a change against judgement (family-1 verdicts)"
- End B: `.abcd/development/brief/05-internals/01-agents.md:236` — "They are each role's own words, not family 1: the ruthless reviewer's `SHIP` shares only its spelling."

The naming register defines the `review` bucket as comparing a change against judgement, with family-1 verdicts, and glosses that family (`{SHIP, NEEDS_WORK, MAJOR_RETHINK}`) as Carmack-style review verdicts. The agent catalog says family 1 assesses a packed lifeboat only, returned by lifeboat-reviewer to `disembark review`. It says the reviewers that judge a change state words of their own: ruthless-reviewer SHIP/FIX FIRST, security-reviewer APPROVE/BLOCK/NEEDS-INPUT. Those words are not family 1. So 'family-1 verdict' names change review on one page and lifeboat review on the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 94.
