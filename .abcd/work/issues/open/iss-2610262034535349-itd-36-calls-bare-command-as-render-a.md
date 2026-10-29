---
schema_version: 1
id: "iss-2610262034535349"
slug: "itd-36-calls-bare-command-as-render-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 531 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-36-memory-unification.md:41"
remedy: "In itd-36 line 41 change 'the universal bare-command-as-render discipline' to 'the bare-command-as-render convention', since 05-intent.md and 08-skills.md say it is common and not universal."
related_intents: [itd-36]
finding_ref: cfk-efbb03d750b2a83f
---

itd-36 calls bare-command-as-render a universal discipline; the intent surface page says it is a common convention, not a universal one.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-36-memory-unification.md:41` — "Bare invocation renders state per the universal bare-command-as-render discipline."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:344` — "Bare-command-as-help is a common abcd convention, not a universal one"

itd-36 justifies the memory command's bare render as following the universal bare-command-as-render discipline. 05-intent.md states outright that bare-command-as-help is a common abcd convention and not a universal one, naming commands whose bare form prints help or a version block instead of a board. Both cannot hold: either every bare command renders state or some do not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 531.
