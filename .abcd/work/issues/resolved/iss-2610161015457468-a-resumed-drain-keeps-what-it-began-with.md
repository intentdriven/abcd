---
schema_version: 1
id: "iss-2610161015457468"
slug: "a-resumed-drain-keeps-what-it-began-with"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Let a running drain take a new --pace and --sub-agents, re-admit handed-back issues whose records changed since, and clear finished handed-back runs"
related_intents: [itd-2610161016401115]
resolution: "A running drain takes a new --pace, --sub-agents and --fix-rounds and applies them from the next lane and the lane in progress's next stage; a handed-back issue whose record blob changed on origin is judged afresh, and a handed-back run no longer passes its issue over"
impact: additive
resolved_by:
  intent: "itd-2610161016401115"
  spec: "spc-2610161044461448"
  commit: "6add08b1918ff140baeef88e1bfa30477b67955a"
---

A resumed drain keeps what it began with for its whole life: its stored pace and sub-agents cannot be changed (--pace is refused while it runs), the issues it handed back stay excluded until it ends, and an issue with an old handed-back run is passed over with 'drive or finish that run' for as long as that run directory exists.
