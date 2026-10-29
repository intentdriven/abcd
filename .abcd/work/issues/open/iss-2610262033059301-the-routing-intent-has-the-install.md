---
schema_version: 1
id: "iss-2610262033059301"
slug: "the-routing-intent-has-the-install"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 358 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:101"
remedy: "In the routing intent's scope bullet and press release say the install states the proposal in counts in one question and writes the full table, readable and editable row by row, on consent, as 01-ahoy.md and commands/ahoy.md describe."
related_intents: [itd-2609170822093401]
finding_ref: cfk-c14c99b8068308e9
---

The routing intent has the install render the proposal, a table with one row per agent, before consent; the ahoy page states the proposal in counts and names no agent.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:101` — "Rendering the proposal and writing it only on consent, at machine level"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:765` — "the proposal said in counts (how many agents, how many at each tier, their fan-out bounds), naming no agent so the question fits"

itd-2609170822093401 defines the proposal as a table with one row per agent, and has the install render it and write it only on consent. The ahoy page's oracle-routing offer states the proposal in counts (agents, agents at each tier, fan-out bounds) and names no agent, so that the question fits. The person therefore consents to a table whose rows they are not shown. The two documents describe different things being consented to.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 358.
