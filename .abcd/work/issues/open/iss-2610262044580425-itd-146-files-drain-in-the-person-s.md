---
schema_version: 1
id: "iss-2610262044580425"
slug: "itd-146-files-drain-in-the-person-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1243 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168"
remedy: "none (filed automatically)"
related_intents: [itd-146, itd-82]
finding_ref: cfk-7ba830f55bde9a62
---

itd-146 files drain in the person's records group; itd-82 says drain joins the person's list only when itd-82 ships

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168` — "records (`capture`, `intent`, `spec`, `decide`, `build`, `drain`, `memory`)"
- End B: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:74` — "The drain joins the person's command list when this intent ships (ruling BX1 kept)."

itd-146's ruled group titles put drain in the person's records group of the default help, beside capture, intent, spec, decide, build and memory. itd-82, still planned, says the drain joins the person's command list only when itd-82 ships, which means it is not on that list now. The two records give incompatible placements for the same verb in the same help list.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1243.
