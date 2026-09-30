---
schema_version: 1
id: "iss-2609300839021188"
slug: "twelve-shipped-surfaces-have-no-brief-chapter-under"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/release/doc-fidelity-backlog.json"
remedy: "Write the missing brief coverage: a 04-surfaces chapter for abcd rules (the loader's rule inspection verb, whose contract lives in 05-internals/03-configuration.md) and one for abcd spec with its close sub-verb (whose behaviour 05-intent.md describes in prose), each carrying the generated appendix; and, for each agent, a code-span naming in the chapter of the verb that delegates to it (cold-reading-* in 23-reading.md, graveyard-interpreter and press-release-composer in 02-disembark.md) or a chapter of its own for the stand-alone reviewers; remove each entry from .abcd/development/release/doc-fidelity-backlog.json in the same change, which the gate then requires. Grounds: adr-5 (the brief is the shipped state) and the gate's own coverage rule; it is shown wrong if a chapter already names one of them in a form the rule misses."
resolution: "Every shipped surface is named by a 04-surfaces chapter: the four cold-reading agents in 23-reading.md, press-release-composer and graveyard-interpreter in 02-disembark.md, and abcd rules, abcd spec, abcd spec close and the three stand-alone reviewers in the register README.md. The recorded backlog is removed, since the gate reads an absent file as empty."
impact: fix
resolved_by:
  commit: "d9c67b639"
---

Twelve shipped surfaces have no brief chapter under 04-surfaces/ naming them: the verbs abcd rules, abcd spec and abcd spec close, and the agents cold-reading-comparative, cold-reading-detection, cold-reading-entailment, cold-reading-widening, graveyard-interpreter, press-release-composer, ruthless-reviewer, security-reviewer and sota-researcher. The doc-fidelity gate's layer 1 (itd-60) found them on its first run over this repository; they are recorded in the gate's baseline so spec close stays usable, which reports them on every run and refuses nothing for them.

## Grounds

- pursued: abcd docs fidelity --report shows 156 of 156 shipped surfaces named and 0 in the backlog; it is shown wrong if that report names any surface as uncovered or backlogged, or if a chapter sentence added here states something the code does not do.
