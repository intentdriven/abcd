---
schema_version: 1
id: "iss-2610262037409579"
slug: "itd-82-s-max-caps-issues-attempted-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 740 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-82-drain-ledger-triage.md:50"
remedy: "In itd-82's Ceilings bullet, say --max caps lanes opened and that the drain runs until nothing eligible is left by default, with no --until-empty flag (that flag belongs to build next)."
related_intents: [itd-82]
finding_ref: cfk-a65dcb9b54783fce
---

itd-82's --max caps issues attempted and names --until-empty as the drain default's flag; 35-drain's cap counts lanes opened, and the drain has no --until-empty

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:50` — "`--max <n>` caps issues attempted and `--until-empty` is the default's own name"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:254` — "When the drain has opened as many lanes as its cap, it ends and says so."

In itd-82 --max counts issues attempted and --until-empty is the name of the default. In 35-drain the drain's cap ends the drain once it has opened that many lanes. Issues that the rule or the host judgement hand back open no lane, so they do not count toward that cap. The shipped flag list in the 35-drain appendix has --max and no --until-empty. So one flag name stands for two different counts, and the other name has no flag behind it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 740.
