---
schema_version: 1
id: "iss-2610262034012499"
slug: "itd-25-grounds-dredge-s-bare-status-help"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 463 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:34"
remedy: "In itd-25 replace 'the universal convention' with 'the common convention' and say bare `/abcd:dredge` renders state, per the Bare invocation section of 04-surfaces/README.md."
related_intents: [itd-25]
finding_ref: cfk-9feb95f02b831800
---

itd-25 grounds dredge's bare status+help in 'the universal convention'; the intent surface says bare-command-as-help is common, not universal

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:34` — "bare invocation = status+help only per the universal convention"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:344` — "Bare-command-as-help is a common abcd convention, not a universal one"

itd-25 justifies its bare invocation as status and help per the universal convention. The intent surface states that bare-command-as-help is a common abcd convention and not a universal one, naming verbs whose bare form prints a plain block or help instead. The convention cannot be both universal and not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 463.
