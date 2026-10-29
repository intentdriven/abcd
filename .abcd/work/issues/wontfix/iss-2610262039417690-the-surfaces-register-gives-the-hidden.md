---
schema_version: 1
id: "iss-2610262039417690"
slug: "the-surfaces-register-gives-the-hidden"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 879 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:276"
remedy: "none (filed automatically)"
related_intents: [itd-2610180912189125]
finding_ref: cfk-13caaee5c3e4de5b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038139122. Same End A line and same missing entrypoints; itd-2610180912189125 is the intent that added the stop check."
duplicates: [iss-2610262038139122]
---

The surfaces register gives the hidden hook verb six entrypoints, while the away-mode intent adds a Stop hook that runs `abcd hook stop`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:276` — "Hidden from `--help`: six host hook entrypoints, live-wired from `hooks/hooks.json`."
- End B: `.abcd/development/intents/shipped/itd-2610180912189125-a-session-told-to-work-on-while-its.md:101` — "A plugin Stop hook asks `abcd hook stop`."

04-surfaces/README.md's operator-internal table gives `hook` exactly six entrypoints (prompt-router, prompt-router-reset, session-end, subagent-stop, session-start, question-answered) and says the two tables together name every verb the binary registers. itd-2610180912189125 shipped a plugin Stop hook that asks `abcd hook stop`, plus a StopFailure hook. Both cannot hold: either the hook verb has more than six entrypoints or the stop check is not one of them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 879.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038139122. Same End A line and same missing entrypoints; itd-2610180912189125 is the intent that added the stop check.
