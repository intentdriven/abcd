---
schema_version: 1
id: "iss-2610311339293870"
slug: "abcd-land-record-review-refuses-a-return"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "landing PR #1291 under away mode"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/review.go"
remedy: "Say in land --help and commands/land.md that the first non-blank line under ### Verdict begins with the verdict word (SHIP, APPROVE, FIX FIRST, BLOCK, NEEDS-INPUT), with the head's nine characters anywhere in the return, and give one example line."
---

abcd land --record-review refuses a return whose first line under ### Verdict does not begin with the verdict word (internal/core/landing/review.go reviewVerdict), so '**977fb39cd: SHIP**' is refused as 'not one a validator writes', yet land --help says only that each return names the head's first nine characters under one ### Verdict heading. A host that briefs its reviewers from the help, as happened on 2026-10-31, gets passing reviews refused and has to ask each reviewer to re-emit.
