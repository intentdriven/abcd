---
schema_version: 1
id: "iss-2610262031588576"
slug: "itd-146-places-a-changelog-verb-in-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 204 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:169"
remedy: "none (filed automatically)"
related_intents: [itd-146]
finding_ref: cfk-ef1c1b3da5eb0c6d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. helpgroups.go has no changelog placement and the stale-usage table answers the retired name with launch --dry-run; the README says changelog is in neither block."
---

itd-146 places a changelog verb in the root help's agent block; 04-naming's register of exempt surfaces, which says every exemption is on the record, does not list it.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:169` — "`changelog` sits in the agent block"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:56` — "The remaining surfaces are exempt for the plain reason that no maritime cognate"

itd-146's decision 3 says '`changelog` sits in the agent block' of the root help, and the intent keeps changelog a top-level verb rather than folding it under launch. 04-naming introduces its list with 'The remaining surfaces are exempt for the plain reason that no maritime cognate' adds meaning, and says each exemption is registered so that it is a decision rather than an omission; changelog is not in the list. Either the register is incomplete or the help names a verb the register does not know.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 204.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. helpgroups.go has no changelog placement and the stale-usage table answers the retired name with launch --dry-run; the README says changelog is in neither block.
