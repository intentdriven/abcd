---
schema_version: 1
id: "iss-2609012039114508"
slug: "install-yes-emits-interactive-value-prompt-on-stdout"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
---

Observation from the lane LA assessment (GHSA-4q78 reproduction): `ahoy install --yes` with missing config values still emitted an interactive value prompt ("visibility (private/public) []:") on stdout, then proceeded as partial when stdin was empty. `--yes` approves categories, not values, so the partial outcome may be by design, but a prompt landing on stdout under a non-interactive flag with no TTY is worth a look: a scripted install reads it as output, and the prompt is the one line in the run that is not a receipt. Not fixed in this run.
