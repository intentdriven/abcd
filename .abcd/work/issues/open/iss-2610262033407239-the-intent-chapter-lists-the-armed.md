---
schema_version: 1
id: "iss-2610262033407239"
slug: "the-intent-chapter-lists-the-armed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 428 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:502"
remedy: "Replace the hand-typed armed-rule enumeration at 05-intent.md line 502 with a pointer to .abcd/record-lint.json (the armed set, per 06-lint.md), or move it into a generated gated region; keep per-rule prose only where it states a rule's own meaning."
finding_ref: cfk-6231d5016eb62a52
---

The intent chapter lists the armed record-lint rules and their severities by hand, while the lint contract says any literal list of rules belongs in a generated, gated region.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:502` — "The armed record-lint rules that bear on the intent tree are `intent_lifecycle`"
- End B: `.abcd/development/brief/05-internals/06-lint.md:14` — "Any literal enumeration of rules in the record belongs in a generated, gated region"

06-lint gives the armed rule set to the JSON configs, and says a literal list of rules in the record belongs only in a generated, gated region, never in a hand-kept table. 05-intent keeps its own hand-typed list of the armed record-lint rules that bear on the intent tree, with their severities. The same ground is covered twice, and the hand-kept copy can drift from the config.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 428.
