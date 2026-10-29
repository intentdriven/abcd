---
schema_version: 1
id: "iss-2610262044589669"
slug: "itd-146-keeps-rules-and-spec-listed-so"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1242 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:76"
remedy: "none (filed automatically)"
related_intents: [itd-146, itd-2610090831227812]
finding_ref: cfk-eab187e0903a532e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later intent names the supersession in its own audit notes; helpgroups.go lists rules and spec with the agents, and the marker block still tells repositories to run abcd rules."
---

itd-146 keeps rules and spec listed so --help matches the marker block; itd-2610090831227812 makes rules and spec agent-only

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:76` — "Hiding them would make `--help` contradict the `ahoy`-installed marker block, which tells every managed repository to run `abcd rules`."
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:97` — "report, spec and rules are agent-only"

itd-146 keeps rules and spec in the listed help, on the ground that hiding them would make --help contradict the ahoy-installed marker block, which tells every managed repository to run abcd rules. itd-2610090831227812 rules report, spec and rules agent-only, which in that intent means hidden from the person's menu and from the matching abcd --help list. It does exactly the hiding itd-146 says creates the contradiction, and it does not address the marker block. Both cannot hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1242.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later intent names the supersession in its own audit notes; helpgroups.go lists rules and spec with the agents, and the marker block still tells repositories to run abcd rules.
