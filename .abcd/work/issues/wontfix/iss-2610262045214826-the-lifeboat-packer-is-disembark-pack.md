---
schema_version: 1
id: "iss-2610262045214826"
slug: "the-lifeboat-packer-is-disembark-pack"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1269 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-24-reflect-command.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-24, itd-88]
finding_ref: cfk-470c10355b191495
refines: [iss-2609091956001547]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/disembark.md runs disembark pack with the repository and destination."
---

The lifeboat packer is `disembark pack <repo> <path>` in itd-24 and `disembark <repo> to <dest>` in itd-88

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-24-reflect-command.md:55` — "`/abcd:disembark pack <repo> <path>` packs"
- End B: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41` — "The packer (`disembark <repo> to <dest>`), embark, and the round-trip"

itd-24 has `/abcd:disembark pack <repo> <path>` pack the voyage's retrospectives into the lifeboat. itd-88 names the packer `disembark <repo> to <dest>`. These are two invocation shapes, a `pack` sub-verb against a positional `to`, for one packing command.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1269.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/disembark.md runs disembark pack with the repository and destination.
