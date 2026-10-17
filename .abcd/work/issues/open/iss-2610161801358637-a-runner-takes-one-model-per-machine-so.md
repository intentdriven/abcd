---
schema_version: 1
id: "iss-2610161801358637"
slug: "a-runner-takes-one-model-per-machine-so"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/config.go"
remedy: "Extend the existing per-invocation route override (--route <agent>=<tier>[@<connection>]) so one invocation can name a runner and the model it uses for a role, on the person's own layer only (their flag, never a repository file), checked against the provider allowlist like any route and recorded in the route receipt; the machine file's model stays the default when nothing overrides it. One override, not a second one."
related_intents: [itd-2610161733316518]
---

A runner takes one model per machine, so a model matrix cannot be configured: internal/core/runner/config.go keeps a single Model per enabled runner, readable only from the machine layer, and roles name runners rather than models. Building one intent with several opencode-served models, or reviewing it with several runner-served reviewers, would mean rewriting the person's machine file between copies, and two copies could never run at once. itd-2610161733316518 needs it.
