---
schema_version: 1
id: "iss-2610262036483439"
slug: "itd-60-names-the-shipped-docs-currency"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 675 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-60]
finding_ref: cfk-283e1f9fc8ecbe10
---

itd-60 names the shipped docs-currency lint `abcd docs lint`; the lint chapter says the docs verb's former check sub-verb is an unknown command

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:44` — "The docs and site verbs' former check sub-verbs are unknown commands"
- End B: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:151` — "`abcd docs lint` — change-narration, resolvable cross-links, no stray root documents"

itd-60 calls the docs-currency lint `abcd docs lint` and says it 'runs everywhere already'. The lint chapter makes that lint the docs target of the one lint (`abcd lint docs`) and says the docs and site verbs' former check sub-verbs are unknown commands. So the spelling itd-60 gives is the one the binary refuses: one surface, two command names, and only one of them resolves.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 675.
