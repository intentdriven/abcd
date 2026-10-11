---
schema_version: 1
id: "iss-218"
slug: "record-tier-tool-naming-convention"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): Should a rule (a principle, or a record-lint check over .abcd/work/ and the intents) keep external tool names out of commitment records until their adoption is decided? No principle or lint carries it at v0.11.1; the harness docs-lint rules reach only docs/ and README."
remedy: "Waits on the ruling on a tool-name rule for commitment records: if adopted, add a principle under .abcd/development/principles/ stating that an external tool is named only in a dated research note until its adoption is decided, and a record-lint banned_tokens family candidate/* whose entries are the tools a research note is still evaluating (each removed when adoption is decided), with record-lint's roots widened to .abcd/work/ for it; if declined, move the record to wontfix naming the harness docs-lint rules on docs/ and README as the chosen boundary. Prove an adoption with a record-lint test flagging a candidate name in an open issue and passing it in a research note."
---

commitment records (issue ledger, DECISIONS, intents) must not name an external tool before its adoption is decided; names live in the dated research note. Convention exists only as maintainer guidance + docs-lint harness rules scoped to docs/ and README; the working record has no principle file or record-lint rule enforcing it

## Remedy grounds (2026-09-29)

- Why: the record's convention needs a list of undecided tools to be lintable at all; a per-candidate banned token is the existing mechanism (.abcd/record-lint.json), and the ruling is unanswered, so both answers are stated.
- Rejected: seeding the rule from the harness/* patterns in .abcd/docs-lint.json, which name adopted hosts the record legitimately discusses; widening the roots also touches the owed question in iss-2608271804494611.
