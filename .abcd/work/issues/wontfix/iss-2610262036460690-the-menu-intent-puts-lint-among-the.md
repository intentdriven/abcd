---
schema_version: 1
id: "iss-2610262036460690"
slug: "the-menu-intent-puts-lint-among-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 672 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:94"
remedy: "none (filed automatically)"
related_intents: [itd-2610090831227812]
finding_ref: cfk-26f337e78febb84e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. commands/lint.md declares block agents as ruled; the chapter's sentence says the rule set is not gated and a human may type it, which does not make it a command people use in practice."
---

The menu intent puts lint among the commands no person types; the lint chapter says bare lint runs when a human types it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:94` — "Agent-only: lint, reflect, inbox, implement, history, site, docs, scribe, identity, guard, hook and statusline."
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:145` — "onboarding and runs when a human types it"

The intent puts lint on the agent-only list, under the ruling that only commands no person types leave the person's menu. The lint chapter says no gate invokes bare `abcd lint`, and that its rule set backs onboarding and runs when a human types it. A command a human types is a person's command by the intent's own rule, so it cannot be on the agent-only list.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 672.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. commands/lint.md declares block agents as ruled; the chapter's sentence says the rule set is not gated and a human may type it, which does not make it a command people use in practice.
