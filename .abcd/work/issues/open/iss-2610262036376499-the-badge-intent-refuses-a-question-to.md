---
schema_version: 1
id: "iss-2610262036376499"
slug: "the-badge-intent-refuses-a-question-to"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 662 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609212130146198-the-status-line-badge-is-true-at-every.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-2609212130146198]
finding_ref: cfk-add309ce56add5cc
---

The badge intent refuses a question to the human while the mode reads managed; the guard chapter says abcd's question is never refused on the mode.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2609212130146198-the-status-line-badge-is-true-at-every.md:40` — "a question to the human through the host's question tool is refused while the mode reads managed"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:183` — "In a repository abcd manages, abcd's question is never refused on the mode"

The badge intent's guard refuses any question put through the host's question tool while the mode reads managed, and leaves naming the addressee to the agent, since deriving it from the question's text is out of its scope. The guard chapter says that in a managed repository abcd's question is never refused on the mode. Instead the gate derives the state from the chip's role and sets it, and an unchipped question asked while the mode reads managed runs unchecked. One gate cannot both refuse on the managed mode and never refuse on it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 662.
