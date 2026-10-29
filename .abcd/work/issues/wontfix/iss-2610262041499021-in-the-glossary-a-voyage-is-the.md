---
schema_version: 1
id: "iss-2610262041499021"
slug: "in-the-glossary-a-voyage-is-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1027 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/voyage.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-90f2d5185a377cbc
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same later ruling (adr-35); shipped intents are not rewritten to follow a later vocabulary."
---

In the glossary a voyage is the operations namespace of lifeboat runs; itd-34 calls an application project a voyage.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/voyage.md:4` — "The operations namespace at `~/.abcd.noindex/voyage/<source-root-sha>/`"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21` — "an application `voyage` (e.g., a macOS app under abcd)"

The voyage entry fixes the term as the operations namespace at ~/.abcd.noindex/voyage/< source-root-sha>/, which contains no project work. itd-34 calls a macOS app under abcd 'an application voyage' and its discipline shape 'voyage-agnostic', which uses the project-lifecycle sense the glossary records as retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1027.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same later ruling (adr-35); shipped intents are not rewritten to follow a later vocabulary.
