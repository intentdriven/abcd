---
id: dcn-2607140000000010
slug: rejected-a-super-reviewer-verb-gated-on
date: 2026-07-14
---
2026-07-14 (review): REJECTED a "super-reviewer" verb gated on a second model
backend. Its motivating rationale does not survive: self-preference under genuine
authorship is weak or absent (arXiv 2606.20093, gap -5.1pp, CI crosses zero), and
the bias that DOES exist on code is a context-framing artefact, not model identity
(arXiv 2603.04582). CHOSEN instead: a fresh-context, off-policy, same-model
reviewer — re-present the diff in a new session as an artefact of unknown
authorship. Captures the only measured debiasing effect, costs nothing, needs no
second subscription, and removes the two-tier UX problem entirely. A second model,
when present, is used for disagreement-as-triage per adr-25 asymmetric trust and
itd-81 — NEVER a panel or a vote (nine judges = 2.18 effective votes; on code,
consensus underperforms the naive baseline — the "popularity trap", arXiv 2510.21513).
