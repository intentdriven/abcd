---
schema_version: 1
id: "iss-2610262034492769"
slug: "itd-111-has-abcd-version-show-install"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 525 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-111]
finding_ref: cfk-68d8900b7f539d15
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-2609212130136102 says --version replaces `abcd version` and the check moves to `update --check`; shipped intents are not rewritten."
---

itd-111 has abcd version show install mode and vintage; the board chapter says version is a retired name and the report is now a flag on the root.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md:23` — "`abcd version` and `abcd ahoy` always show install"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:458` — "`version`, whose page retired when the board took the version"

itd-111's press release names the abcd version verb, beside ahoy, as the surface that always shows install mode and vintage. 08-abcd states that the full answer is a flag on the root rather than a verb, and lists version among names retired with no alias, whose page retired when the board took the version, so typing it is answered with what replaced it. One report carries two names, one of which no longer runs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 525.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-2609212130136102 says --version replaces `abcd version` and the check moves to `update --check`; shipped intents are not rewritten.
