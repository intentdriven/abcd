---
schema_version: 1
id: "iss-2610262033191807"
slug: "itd-88-names-the-packer-disembark-repo"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 390 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-88]
finding_ref: cfk-4ebc859ea2737d8c
---

itd-88 names the packer `disembark <repo> to <dest>`; 02-disembark makes bare disembark a read-only listing and packing the `pack` sub-verb

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41` — "The packer (`disembark <repo> to <dest>`), embark, and the round-trip"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:44` — "Bare `/abcd:disembark` prints the sub-verb list and flags, and mutates nothing."

itd-88 writes the packer as the bare verb with a `to` keyword, `disembark <repo> to <dest>`. 02-disembark says bare `/abcd:disembark` prints the sub-verb list and flags and mutates nothing, and lists packing as the `pack` sub-verb, which takes two positional paths. The bare form cannot both pack a lifeboat and mutate nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 390.
