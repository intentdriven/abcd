---
schema_version: 1
id: "iss-2610262032268272"
slug: "embark-s-refresh-audit-modifier-diffs"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 269 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:248"
remedy: "none (filed automatically)"
finding_ref: cfk-b138d608193740e6
---

Embark's refresh-audit modifier diffs against a disembark-time audit; disembark produces a review verdict and no audit

A sequencing impossibility (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:248` — "would re-run the oracle product audit against the current lifeboat content and report the drift against the disembark-time audit"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:205` — "the `lifeboat-reviewer` review on the run's outputs returns a registered verdict with specific findings rather than vague approval"

Embark's design-target refresh-audit modifier would re-run the oracle product audit and report drift against the disembark-time audit. Disembark's pipeline has no audit step: its judgement over the pack is the lifeboat-reviewer's review returning a registered verdict. The modifier depends on an artefact disembark, as written, never produces, or names the review by another verb's word.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 269.
