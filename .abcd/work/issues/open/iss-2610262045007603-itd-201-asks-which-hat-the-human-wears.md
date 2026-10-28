---
schema_version: 1
id: "iss-2610262045007603"
slug: "itd-201-asks-which-hat-the-human-wears"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1245 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:64"
remedy: "none (filed automatically)"
related_intents: [itd-201, itd-2609212130146198]
finding_ref: cfk-274233f0e8a96b73
---

itd-201 asks which hat the human wears before the mode is set; itd-2609212130146198 refuses every question while the mode reads managed

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:64` — "When the agent does not know which hat the human wears, the first question asks that"
- End B: `.abcd/development/intents/shipped/itd-2609212130146198-the-status-line-badge-is-true-at-every.md:40` — "a question to the human through the host's question tool is refused while the mode reads managed"

itd-201 has an agent that does not know which hat the human wears ask that as its first question, and only then record the answer in the itd-200 mode. itd-2609212130146198's guard refuses any question through the host's question tool while the mode still reads managed, which is the state until the agent has named the addressee. The first question itd-201 calls for therefore cannot be put through the question tool in the order itd-201 gives.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1245.
