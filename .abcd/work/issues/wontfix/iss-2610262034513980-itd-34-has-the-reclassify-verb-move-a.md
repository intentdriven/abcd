---
schema_version: 1
id: "iss-2610262034513980"
slug: "itd-34-has-the-reclassify-verb-move-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 528 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-34-three-intent-kinds.md:61"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-a4eae4cd83da88e3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/reclassify.go moves no file on a kind change, writes no discipline and refuses a shipped record's kind change, matching 05-intent.md and itd-34's own criteria; the kind proposal and pre-commit shape text sit under the re-scope banner."
---

itd-34 has the reclassify verb move a draft into disciplines/; the intent surface page says no reclassify writes a discipline and a move into disciplines/ is refused.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:61` — "Moves the file between directories as the new kind dictates"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:301` — "no reclassify writes a discipline, on any shelf"

itd-34 scopes the reclassify subcommand to late kind changes, a draft realised to be a discipline among them, and says it moves the file between directories as the new kind dictates. 05-intent.md says no reclassify writes a discipline on any shelf, that any move into disciplines/ is refused with nothing written, and that a kind change on a draft or planned record leaves the shelf where it is. The two cannot both describe the shipped verb. The same intent also says the plan step proposes a kind for the user to confirm and that the shape role runs continuously from a pre-commit hook, where the surface page says the shipped plan step neither infers nor asks for a kind and no hook invokes the shape scan.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 528.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/reclassify.go moves no file on a kind change, writes no discipline and refuses a shipped record's kind change, matching 05-intent.md and itd-34's own criteria; the kind proposal and pre-commit shape text sit under the re-scope banner.
