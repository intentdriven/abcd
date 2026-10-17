---
schema_version: 1
id: "iss-2610161015422306"
slug: "the-drain-s-host-judgement-has-no-route"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/judgement.go"
remedy: "Route the judgement through the model routing the build roles use, and keep a hosted session's drain state and copy key across a restart, or accept --take-over on drain"
related_intents: [itd-2610161016401115]
resolution: "The drain's judge is a drain-judge role on the runner routing, answered and taken in one move with abcd computing the remedy digest (ffda8dd52), and abcd drain --take-over moves this person's own reservations from a lost copy as the drain reaches them, the pass-over naming the flag"
impact: additive
resolved_by:
  intent: "itd-2610161016401115"
  spec: "spc-2610161044461448"
  commit: "d7d49ff2fb6904d241fb9705f5983a4191baa425"
---

The drain's host judgement has no route to a command-line model, so an unattended drain needs a host agent looping; in a plain terminal a person writes the JSON answer, sha256 digest included, by hand; and a hosted session that restarts loses .abcd/.work.local, so its new copy key reads its own reservations as another copy's and the advice names --take-over, which drain does not accept.
