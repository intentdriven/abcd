---
schema_version: 1
id: "iss-2610262034417109"
slug: "itd-122-says-surface-coverage-checks"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 515 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-122-the-registry-cannot-wave-its-hands-every.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-122]
finding_ref: cfk-cd0bb34b91995ed3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-122 acceptance and audit notes record exemptFromCobra for the bare command in internal/core/lint/subverbs.go; the chapter's sentence restates that exemption, not a departure from it."
---

itd-122 says surface_coverage checks every surface file's sub-verb rows both ways; the board chapter says its table is compared with the command tree in neither direction.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-122-the-registry-cannot-wave-its-hands-every.md:20` — "now checks every row against the binary's own command tree, both ways"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:27` — "the rule compares its table with the command tree in neither direction"

itd-122's press release states that every surface file under 04-surfaces/ carries a sub-verb table and that surface_coverage checks every row against the command tree both ways, so an unlisted shipped sub-verb and a row for an unregistered one each fail the build. 08-abcd says that, as the bare command's chapter, the rule compares its table with the command tree in neither direction, its row being covered by the surfaces index instead. One surface file is exempt from a check the intent describes as universal.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 515.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-122 acceptance and audit notes record exemptFromCobra for the bare command in internal/core/lint/subverbs.go; the chapter's sentence restates that exemption, not a departure from it.
