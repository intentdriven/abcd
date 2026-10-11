---
schema_version: 1
id: "iss-2609301557141123"
slug: "the-claude-runner-takes-the-model-its"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/claude.go"
remedy: "Bound and shape the reported model as sessionRe bounds a session id (a plain id of at most 128 characters, a bracketed context suffix admitted), and refuse any other as an unparsable answer, so the route falls back and the fallback is recorded rather than the model written into the state."
resolution: "The init event's model is held to modelRe (bounded, plain, a bracketed suffix admitted); any other is an unparsable answer, so the route falls back and is recorded and the model never reaches state.json."
impact: fix
resolved_by:
  commit: "70d49494f"
---

The claude runner takes the model its harness's init event reports unbounded and unshaped into the answer (internal/core/runner/claude.go parseClaude), and the loop writes it into state.json through the route record; a model string over 4 MiB pushes state.json past its read bound, so every later read refuses and the run is bricked, and control bytes travel into the record verbatim.

## Grounds

- pursued: a harness reporting a 5 MiB model or one carrying control bytes falls back with an unparsable reason and the run's state stays readable with no receipt carrying it; a real id such as claude-opus-4-6[1m] refused, or a state.json holding the odd model, would show it wrong
