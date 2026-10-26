---
schema_version: 1
id: "iss-2610262036554739"
slug: "the-build-chapter-refuses-any-key-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 684 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:231"
remedy: "none (filed automatically)"
finding_ref: cfk-c72b79054080fdbd
---

The build chapter refuses any key under `pace` the loop does not read, while the drain reads two more keys from that namespace

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:231` — "the reader claims the `pace` namespace, so a key under it the loop does not read is refused rather than ignored"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:233` — "both bounds are read from the pace's layers (`pace.stuck_refusals`, `pace.stuck_minutes`) at every move"

34-build resolves four pace keys (work_minutes, pause_minutes, sub_agents, fix_rounds) and says the reader claims the `pace` namespace, refusing any key under it the loop does not read. 35-drain reads its stuck bounds from `pace.stuck_refusals` and `pace.stuck_minutes` in the pace's layers. As written, a repository that sets a stuck bound for its drain holds a key under `pace` that the build chapter says is refused.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 684.
