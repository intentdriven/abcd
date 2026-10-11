---
schema_version: 1
id: "iss-334"
slug: "intents-readme-md-teaches-the-v0-6-0"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/development/intents/README.md"
resolution: "intents/README.md swept intent review->intent audit and intent-fidelity-reviewer->intent-auditor at the eight current-state sites; historical struck-through row left"
impact: internal
---

intents/README.md teaches the v0.6.0-retired intent review verb and the intent-fidelity-reviewer agent (retired to intent audit / intent-auditor by adr-40); the intents store front door documents an invocation that exits non-zero and an agent absent from the tree, and its template block is copied into new intents while contradicting create.go

## Evidence

- `.abcd/development/intents/README.md:55,113,114,115,133,138,218,315` -- intent review, intent review ingest, intent-fidelity-reviewer.
- cli.go retiredSubverbs refuses intent review (successor audit, adr-40/spc-28); agent file is agents/intent-auditor.md; create.go:260 emits the intent-auditor stub the :218 template block contradicts.

## Refuter verdict -- CONFIRMED (substantive, doc-only)

All eight sites present-tense; the one legitimately historical line (:262, struck-through, dated) is correctly excluded. spc-28's Not-swept list does not include this file; it fell between spc-28's two lists and no commit swept it. No banned_tokens entry catches the spellings. Fix: sweep the eight lines review->audit / intent-fidelity-reviewer->intent-auditor, leaving :262.
