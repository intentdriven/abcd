---
schema_version: 1
id: "iss-2610262043574593"
slug: "itd-146-lists-memory-among-the-person-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1173 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168"
remedy: "none (filed automatically)"
related_intents: [itd-146, itd-2610091918433290]
finding_ref: cfk-a2340699146480a7
---

itd-146 lists `memory` among the person's records verbs; itd-2610091918433290 says `/abcd:memory` is an agent's command for memory notes

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168` — "records (`capture`, `intent`, `spec`, `decide`, `build`, `drain`, `memory`)"
- End B: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:58` — "`/abcd:memory` is an agent's command."

itd-146 groups `memory` with `capture`, `intent` and the other records verbs that a person reads in `--help`. itd-2610091918433290 says a verb split freed the name `memory` and merged its old job into `library`, and that `/abcd:memory` is now an agent's command for memory notes. The name `memory` therefore stands for two different commands, filed under different audiences.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1173.
