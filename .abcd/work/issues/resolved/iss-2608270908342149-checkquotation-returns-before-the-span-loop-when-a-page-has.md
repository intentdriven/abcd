---
schema_version: 1
id: "iss-2608270908342149"
slug: "checkquotation-returns-before-the-span-loop-when-a-page-has"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/memory/lint.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Does checkQuotation's span check run on a hashless page (MQ001 on 160-word spans)?"
resolution: "the inconsistency is gone at the base: the 06-lint.md sentence it quoted (the denominator-free span half of MQ001 still applies) was removed when b760a67c5 rewrote the lint chapter to describe the shipped code, and the standing record scopes MQ001's span half to a page with an external source (itd-36 ac-4, brief/04-surfaces/07-memory.md); a page with none gets MQ003 naming the skip, which is what checkQuotation does"
impact: internal
resolved_by:
  commit: "b760a67c5"
---

checkQuotation returns before the span loop when a page has no external hashes, so a legitimately hashless page quoting a 160-word contiguous span emits no MQ001 at all, while 06-lint.md states the denominator-free span half still applies — fixing it changes lint output for legitimate pages, so the contract call is a human decision

## Grounds

- pursued: no record now promises MQ001 on a hashless page; a brief, spec or intent restating that promise against checkQuotation's early return would show it wrong
