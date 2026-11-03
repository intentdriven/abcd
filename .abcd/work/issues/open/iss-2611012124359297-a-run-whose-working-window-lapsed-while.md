---
schema_version: 1
id: "iss-2611012124359297"
slug: "a-run-whose-working-window-lapsed-while"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "API-key build run run-2611010929062241, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "When a run's window has elapsed and its pause is zero, let a step that carries --route take the pause and open the next window itself, then hand the role out on the route; or name in the refusal that one step without --route reopens the window."
---

A run whose working window lapsed while its preflight waited in the load gate refuses a step that carries --route: 'refused at route: --route ... routes the ruthless-reviewer, and this step hands no role to an agent: the run's window has elapsed, so the step pauses the run'. With a zero-minute pause the person must step once without the route to reopen the window and again with it, an extra round each time a long queue outlasts a window, and a step without the route that happens to hand out a role gives it to the host for good.
