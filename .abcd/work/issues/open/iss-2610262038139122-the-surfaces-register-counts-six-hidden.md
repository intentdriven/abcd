---
schema_version: 1
id: "iss-2610262038139122"
slug: "the-surfaces-register-counts-six-hidden"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 780 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:276"
remedy: "none (filed automatically)"
finding_ref: cfk-c9f383759a446cd0
---

The surfaces register counts six hidden hook entrypoints; the configuration chapter wires two more, `hook stop` and `hook stop-failure`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:276` — "Hidden from `--help`: six host hook entrypoints, live-wired from `hooks/hooks.json`."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1337` — "mode's stop check (`hook stop`, `hook stop-failure`, spc-2610180914469489) and fire"

04-surfaces/README lists `hook` with six entrypoints (prompt-router, prompt-router-reset, session-end, subagent-stop, session-start, question-answered) and claims its tables name every verb the binary registers. 03-configuration says the `Stop` and `StopFailure` events carry `hook stop` and `hook stop-failure`. Both counts cannot be complete, and the operator-internal table leaves two live entrypoints out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 780.
