---
schema_version: 1
id: "iss-2610222000271964"
slug: "abcd-drain-run-from-a-plain-terminal"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "PT ran abcd drain in a plain terminal after the 2026-10-22 crash"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "when no host session drives the drain, abcd drain launches claude sessions itself through the runner abcd already uses to start agents, one per judgement and lane step, and loops until its cap or the ledger is done"
related_intents: [itd-2610222215485265]
---

abcd drain run from a plain terminal stops at its first move: it reserves the first eligible issue, writes a host-judgement request, and waits for an agent session to answer it, so a person who types abcd drain in a terminal gets no fixes (seen 2026-10-22 20:47 BST on a second machine: 213 eligible, 0 lanes opened, iss-2610120138028442 reserved for 2h and left). The person wants abcd drain to work from the plain terminal by launching claude sessions itself to judge each remedy and drive each lane.

## Grounds

- pursued: typing abcd drain in a plain terminal should work through the ledger with no host session; shown wrong if a terminal drain left for eight hours with the runner set up opens fewer lanes than a host-driven drain in the same time, for a reason other than its cap
