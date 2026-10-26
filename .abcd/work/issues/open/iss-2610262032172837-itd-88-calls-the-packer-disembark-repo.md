---
schema_version: 1
id: "iss-2610262032172837"
slug: "itd-88-calls-the-packer-disembark-repo"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 248 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-88]
finding_ref: cfk-bc52d484ce216dd4
---

itd-88 calls the packer `disembark <repo> to <dest>`; the platform page calls it `disembark pack <source-repo> <dest>`

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41` — "The packer (`disembark <repo> to <dest>`), embark, and the round-trip"
- End B: `.abcd/development/brief/02-constraints/01-platform.md:21` — "`disembark pack <source-repo> <dest>`. There is no in-tree lifeboat home"

itd-88 refers to the packer as `disembark <repo> to <dest>`. 01-platform gives the lifeboat path as `disembark pack <source-repo> <dest>`: a `pack` sub-verb with no `to` keyword. The record holds two invocation shapes for one operation.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 248.
