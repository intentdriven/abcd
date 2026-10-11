---
schema_version: 1
id: "iss-2608231021254712"
slug: "the-relationship-between-the-cli-command"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "session-closeout-audit-2026-08-23"
found_at: ".abcd/development/research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): Is the CLI command tree an engine API free to diverge from the plugin surface's spelling, or must the two keep one spelling? No ADR or DECISIONS entry has ruled it at v0.11.1."
remedy: "Waits on the ruling on whether the CLI tree may diverge from the plugin surface's spelling; record the answer as an ADR either way. If one spelling: surface_coverage refuses a Go verb with no command page of the same name, and a page with no verb, unless the pair is on a declared exception list (changelog, spec and rules today); if free to diverge: each command page declares in its frontmatter the verb path it invokes and surface_coverage joins on that declaration instead of the name. Prove either with a surface_coverage test over a fixture tree holding one undeclared divergence."
---

The relationship between the CLI command tree and the plugin command surface is an open decision with no record that says so, so it is invisible to anyone scanning open work. The 2026-08-22 verb-taxonomy hand-run routed five parts and filed four. The fifth was HELD: whether the CLI tree is an engine API that the flat plugin surface merely calls, so the two need not carry the same spellings. It was not filed because an ADR records a ratified choice and the maintainer had not ruled, which was the right call at the time and is now the problem: nothing prompts the ruling. Its only durable homes are prose. research/notes/2026-08-15-decomposition-calibration.md names it in one HELD line inside a graded entry, and research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md discusses decoupling only as part of the idea it killed, so a reader of the verdict would reasonably conclude the question died with the restructure. It did not. The killed proposal was a noun-verb hierarchy with invented category nouns; the surviving question is narrower and independent of it. What makes it live rather than academic: the two surfaces are ALREADY not one to one, in both directions and by explicit config. Three commands are host-delegated markdown with no Go verb, and changelog, spec and rules are Go verbs with no markdown surface. So decoupling as a principle is precedented and partially in force, and what has never been decided is whether that is a deliberate architecture or an accumulation of exceptions. The verdict's leg 2 and leg 3 supply the evidence either way: iss-161 records that the harness structurally refuses a nested markdown surface, so the plugin side cannot mirror a nested CLI even if wanted; and the strongest recorded objection is that decoupling deletes the three-way name join, on which the surface_coverage sub-verb pass depends, against roughly 109 fenced binary invocations and 18 argument-hint lines under commands/. Deciding it settles whether a future CLI-side change may diverge from the plugin spelling at all, which is a precondition for any grouping, nesting or rename proposal that touches only one side. Not urgent, and deliberately carrying no proposal: this capture exists so the question is answerable from the ledger rather than only from a research note a later reader has no reason to open.

## Remedy grounds (2026-09-29)

- Why: the record carries no proposal by design; the evidence it names (pages with no verb, verbs with no page, and the fenced invocations the three-way name join covers) makes the join the thing either answer must keep.
- Rejected: leaving the answer in the research note, the invisibility this record exists to end.
