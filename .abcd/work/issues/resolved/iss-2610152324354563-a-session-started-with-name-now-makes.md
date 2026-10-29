---
schema_version: 1
id: "iss-2610152324354563"
slug: "a-session-started-with-name-now-makes"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "fix round 4 of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Resolve the root lazily, only on the paths that read the persona list; test that a titled startup runs no git."
refines: [iss-2609091729516940]
resolution: "sessionname.Acts, the gate the session-start hook resolves the repository root behind, now answers whether Start reads the persona list, and no for a startup with a title; Start keeps its own guard, so a titled startup still records the name as the person's and the hook asks git nothing for it."
impact: fix
resolved_by:
  commit: "c219de389816d7292259b3c264de661e43696f22"
---

A session started with --name now makes the session-start hook resolve the repository root (a git call) on its keep path, which only records an empty given entry and needs no repository.

## Grounds

- pursued: a session started with --name resolves no repository root at session start while still being recorded as the person's own; TestActsGatesExactlyThePersonaRead would show it wrong if any payload kind's start read the persona list with Acts false (a root needed but not resolved) or Acts stayed true for a titled startup (git asked for nothing).
