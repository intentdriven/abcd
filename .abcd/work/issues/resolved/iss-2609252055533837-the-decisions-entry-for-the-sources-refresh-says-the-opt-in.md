---
schema_version: 1
id: "iss-2609252055533837"
slug: "the-decisions-entry-for-the-sources-refresh-says-the-opt-in"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/DECISIONS.md"
resolution: "A dated entry appended to .abcd/work/DECISIONS.md corrects the 2026-09-25 sources-refresh entry: the scaffolded template takes a provisional stance on two of the ruling's three questions (opt-in by default; fail open on an unusable opt-in), leaving only how a scaffolded hook finds abcd unanswered, and both stances stand until iss-2609250834251447 is ruled. The earlier entry is left as written, since DA002 refuses any removed line. No gate would have caught it: lint-decisions checks the ledger's shape, not an entry's claims."
impact: internal
resolved_by:
  commit: "fb923e937"
---

The DECISIONS entry for the sources refresh says the opt-in shape decides none of iss-2609250834251447's questions, but the template does take a provisional stance for the pre-commit refresh (opt-in; fail open on an unusable opt-in); the entry should call that half provisional (review2-sources 5).

## Grounds

- pursued: the appended entry states what internal/core/ahoy/defaults/pre-commit does (nothing runs without abcd.sourcesBinary; an unusable setting prints one line and the commit proceeds); a template branch that refuses the commit, or runs a binary without the opt-in, would show it wrong
