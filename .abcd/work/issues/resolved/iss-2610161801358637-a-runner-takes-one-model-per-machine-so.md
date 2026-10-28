---
schema_version: 1
id: "iss-2610161801358637"
slug: "a-runner-takes-one-model-per-machine-so"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/runner/config.go"
remedy: "Extend the existing per-invocation route override (--route <agent>=<tier>[@<connection>]) so one invocation can name a runner and the model it uses for a role, on the person's own layer only (their flag, never a repository file), checked against the provider allowlist like any route and recorded in the route receipt; the machine file's model stays the default when nothing overrides it. One override, not a second one."
related_intents: [itd-2610161733316518]
resolution: "abcd implement step takes --route <role>=runner:<name>[@<provider>/<model>], the route override's runner rung: for that step it runs the role on the named runner and allowlisted model, refusing an unknown or disabled runner and an unlisted model before anything is written, and records the --route in the role's receipt"
impact: additive
resolved_by:
  commit: "cad52f01d"
---

A runner takes one model per machine, so a model matrix cannot be configured: internal/core/runner/config.go keeps a single Model per enabled runner, readable only from the machine layer, and roles name runners rather than models. Building one intent with several opencode-served models, or reviewing it with several runner-served reviewers, would mean rewriting the person's machine file between copies, and two copies could never run at once. itd-2610161733316518 needs it.

## The grammar chosen (2026-10-17)

The override keeps one flag and one parser and gains a runner rung:
`--route <role>=runner:<name>[@<provider>/<model>]`. The `runner:` prefix marks
the rung, and no tier is spelt with a colon, so every route the tier rung
admitted reads as it did. `@` keeps its meaning of where the model comes from:
a provider connection on the tier rung, a provider's model on the runner rung.
Without `@` the runner keeps its machine model. Three alternatives were
rejected. A bare runner name in the tier's place (`implementer=opencode@...`)
puts a runner and a tier in one slot with nothing to tell them apart. The runner
in the connection's place (`implementer=frontier@runner:opencode/...`) needs a
tier that chooses nothing. A separate `--runner` flag is the second override the
remedy rules out. Runners are dispatched only by the build loop's step today, so
the rung is honoured on `abcd implement step`; a delegating verb, which sends its
agent to the harness or a provider, refuses it, and the step refuses a tier route.
