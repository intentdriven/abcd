---
schema_version: 1
id: "iss-2610262042208017"
slug: "itd-88-writes-the-packer-as-disembark"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1063 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-88]
finding_ref: cfk-160d24114622ba31
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same recorded divergence as the other itd-88 packer records."
---

itd-88 writes the packer as `disembark <repo> to <dest>`; the disembark entry writes it as `abcd disembark pack <source-repo> <dest>`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41` — "The packer (`disembark <repo> to <dest>`), embark, and the round-trip"
- End B: `.abcd/development/brief/glossary/core/disembark.md:4` — "`abcd disembark pack <source-repo> <dest>` reads a source repository without writing to it"

itd-88's scope names the packer that comes after the probe and coverage aggregate `disembark <repo> to <dest>`. The disembark glossary entry defines the same act as `abcd disembark pack <source-repo> <dest>`. That is two command forms for one pack operation, so the documents disagree on what to type.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1063.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same recorded divergence as the other itd-88 packer records.
