---
id: adr-54
slug: punctuation-enforcement-stays-mechanical
status: accepted
date: 2026-08-28
supersedes: [itd-141]
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-53]
---

# ADR-54: Punctuation enforcement stays mechanical-only

## Context

The writing-style guide staged its punctuation rules for a mask-aware
docs-lint (itd-141): casing after colons and semicolons, and the em-dash
list-item rule. The em-dash core shipped on 2026-08-28 as the
`punctuation/em-dash-in-list-item` banned token (iss-2608280706531199),
armed after the corpus was swept clean per the guide's warn-then-promote
rule.

An adversarial design review then measured what remained against the corpus
(`docs/` + `README.md`, fence- and inline-code-masked):

- Colon-casing: 0 true positives, 3 false positives even in its narrowest
  scope (list-item pivots) — and all three are the project's own
  lowercase-by-design name opening a definition.
- Semicolon-casing: 0 true positives, 15 false positives — citation
  reference lists and acronyms, where a capital after a semicolon is correct
  English and mechanically indistinguishable from a violation without a
  dictionary.
- The em-dash residual (a wrapped continuation line of a list item): 0
  instances.

The compounding fact: under the guide's promotion rule a check enters as a
`warn` and is promoted to `blocker` only on a clean corpus. A check whose
false-positive class cannot be closed is never corpus-clean at blocker
severity, is therefore never promoted, and therefore never flips its row's
label to machine-enforced — which was itd-141's entire goal. The goal is
unreachable under the repository's own rules, not merely hard.

## Decision

1. **Punctuation enforcement is mechanical-only.** What ships is the
   line-based banned token plus the `<!-- docs-lint: allow -->` escape. No
   dictionary-backed prose lint enters the native linter.
2. **The colon and semicolon casing rules are `review` by nature,
   permanently.** The guide records the reason on the rows themselves.
3. **The continuation-line em-dash residual is accepted as `review`.** Zero
   corpus yield does not warrant machinery (less-but-better;
   guards-prove-themselves). A first observed regression reopens it as a
   fresh intent (recurrence-is-signal).
4. **itd-141 is superseded by this decision** and retires to the
   supersession record.
5. **Full-guide enforcement, if ever wanted, is an adoption, not a
   reimplementation:** a Vale-class prose linter filed as its own intent,
   exactly as itd-141's SOTA section already ruled.

## Alternatives Considered

1. **Ship the casing lints with a lowercase-proper-noun allowlist.**
   Rejected: an open-ended dictionary is the classic prose-lint failure
   mode, and no allowlist closes the semicolon class (any proper noun may
   follow one).
2. **Ship the casing rules as permanent advisory warns.** Rejected: warns
   gate nothing and never flip labels, so they add standing noise without
   moving the intent's goal (the `spelling/*` family is different — its
   warns catch real drift with a bounded escape set).
3. **Build the continuation-line rule now.** Rejected on yield: the corpus
   has zero instances, and the building blocks (fence mask, inline-span
   strip, list-state tracking) remain available the day a regression makes
   it worth a record.

## Consequences

The guide's punctuation rows state final labels with their reasons; the
"staged lints" forward-claim is removed; no row cites itd-141 as a staging
promise. The supersession is recorded on both sides (this ADR's
`supersedes`, the intent's `superseded_by`). The decomposition-calibration
note carries the graded hand-run that led here.
