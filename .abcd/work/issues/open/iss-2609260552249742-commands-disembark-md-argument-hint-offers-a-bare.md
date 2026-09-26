---
schema_version: 1
id: "iss-2609260552249742"
slug: "commands-disembark-md-argument-hint-offers-a-bare"
severity: "minor"
category: "documentation"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/disembark.md"
---

commands/disembark.md argument-hint offers a bare `<source-repo> <dest>` form, but only `abcd disembark pack <repo> <dest>` exists: typing the advertised form fails with `unknown command "<repo>" for "abcd disembark"`. Found by the v0.11.0 brief-surface cross-check (x-006), reproduced by the classifier.
