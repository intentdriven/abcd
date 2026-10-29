---
schema_version: 1
id: "iss-2610262039403534"
slug: "itd-111-says-abcd-version-always-shows"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 877 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-111]
finding_ref: cfk-dd0328497b608c8e
refines: [iss-2609091956001547]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The surfaces README names itd-2609212130136102 as the record that made version a root flag; itd-111 predates it."
---

itd-111 says `abcd version` always shows install mode and vintage; the surfaces register says `version` is a root flag, not a verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md:23` — "`abcd version` and `abcd ahoy` always show install"
- End B: `.abcd/development/brief/04-surfaces/README.md:156` — "`version` is the root's `--version` flag (itd-2609212130136102), not a verb"

itd-111 says `abcd version` and `abcd ahoy` always show install mode and vintage. The surfaces register says `version` is the root's `--version` flag, not a verb, and lists no `version` verb in either of its tables of registered verbs. The intent's claim names a command that the register says the binary does not register.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 877.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The surfaces README names itd-2609212130136102 as the record that made version a root flag; itd-111 predates it.
