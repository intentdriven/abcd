---
schema_version: 1
id: "iss-2610112232319827"
slug: "guard-misses-npx-kill-port"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "guard lane G2a, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Read npx (and npm exec, pnpm dlx, yarn dlx) as a wrapper whose first operand is the program, and add kill-port to the stop-by-holder entry with the same successor; prove with Check tests"
resolution: "kill-port now blocks as stopping by holder does (ruling AS), under a kill-port entry with the successor kill-by-holder and fuser-kill name. npx, npm exec (and npm x), pnpm dlx and yarn dlx are read as wrappers whose first operand is the program, with their documented value flags, and a package spec's version is read off the program name, so a blocker behind a runner is read as the blocker. Proved by TestKillPortIsStoppingByHolder, watched failing first, the entry's fixtures under the substitution-position property, and adversarial corpus lines; the false-positive sweep stays at 25 blocks and 29 warns. The new entry raises the costliest unknown-word shape to about 28.4 work units per byte, so the documented bar moves from 28 to 30."
impact: additive
---

The guard has no entry for `npx kill-port <port>` (and its kin such as `kill-port` installed globally), a common way to stop whatever holds a port, which the product thinker ruled on 2026-10-11 should be blocked like stopping by name; `npx` is also not a wrapper the guard looks through. Found by guard lane G2a on 2026-10-11.
