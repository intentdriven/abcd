---
id: dcn-2608170000000002
slug: readme-bucket-vocabulary-alignment-289
date: 2026-08-17
---
2026-08-17 — README bucket-vocabulary alignment (#289), made unattended
against adr-40's own table rather than by taste: grading delivered work
against acceptance criteria is an *audit* (reality vs a recorded
commitment), so four "review" call sites move to audit wording, agreeing
with the one call site that already said "intent auditor". Two further
calls inside the same passage: (1) "a privacy review" becomes "a
privacy-hygiene lint" — a cross-cutting rule every feature must satisfy is
adr-40's definition of a lint, and abcd's own `privacy-hygiene` emits a
rule id and an exit code, not SHIP/NEEDS_WORK, so the example contradicted
the sentence defining it; (2) "owned by the AI-engineering team" becomes
"run by the AI-engineering team, read by you" — the role table gives the
product thinker the audit bucket, so *owned by* pointed the reader at the
wrong end of the handover and made the one bucket they do own read as
someone else's property. Also corrected a factual gap: cross-cutting rules
do not merely "go straight into the brief", they have a record home in
`intents/disciplines/` (`kind: discipline`; itd-1/5/37/79/81/84 live there
today). Rejected: leaving the prose loose on the grounds that user-facing
writing need not track the code's vocabulary — adr-40 closed the bucket
list precisely because an autonomous run given a criterion will interpret
it, and the README is the first thing both a contributor and an agent read.
