---
schema_version: 1
id: "iss-2611220942578412"
slug: "while-a-person-is-away-bare-abcd-mode"
severity: "nitpick"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
remedy: "Have bare abcd mode show the away state, its end time and the queued count, and bare abcd mode queue list the queued steps one per line, writing nothing."
resolution: "Bare abcd mode prints a second line with the away end time and queued count while an away record holds, and bare abcd mode queue lists the queued steps one per line and writes nothing."
impact: additive
resolved_by:
  commit: "550f79257"
---

While a person is away, bare abcd mode prints only managed, and the steps queued with abcd mode queue can be read only by ending the away mode with abcd mode back; a run cannot list what it has already queued for the person, so it cannot avoid queueing the same step twice or tell the person on return what is waiting without ending the mode.
