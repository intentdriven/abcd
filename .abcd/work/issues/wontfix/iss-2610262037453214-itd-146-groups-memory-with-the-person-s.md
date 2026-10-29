---
schema_version: 1
id: "iss-2610262037453214"
slug: "itd-146-groups-memory-with-the-person-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 746 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168"
remedy: "none (filed automatically)"
related_intents: [itd-146]
finding_ref: cfk-3197ab77c7f81c12
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. helpgroups.go lists memory in the agents block and notes the old memory and source verbs merged into library."
---

itd-146 groups memory with the person's records verbs in help; 37-memory says help lists the memory verb with the agents

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168` — "records (`capture`, `intent`, `spec`, `decide`, `build`, `drain`, `memory`)"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:6` — "the binary's help lists the verb with the agents"

itd-146's ruling on group titles files memory beside capture, intent, spec, decide, build and drain in the person's records group of abcd --help. 37-memory says memory is an agents' command whose page is hidden from a person's menu and which the binary's help lists with the agents. The two documents put the same verb name in different help blocks. If itd-146 meant the verb that existed before the memory and library split, the name now denotes another command.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 746.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. helpgroups.go lists memory in the agents block and notes the old memory and source verbs merged into library.
