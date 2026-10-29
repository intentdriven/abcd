---
schema_version: 1
id: "iss-2610262038113742"
slug: "the-adapters-chapter-says-seam-backend"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 778 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/02-adapters.md:28"
remedy: "In 02-adapters.md (and the same sentence in 04-universal-patterns.md) say the seam.backend selectors are design: only oracle.backend is read, it records a choice and routes nothing, and spec.backend, run.backend and history.backend are staged keys."
finding_ref: cfk-a46f8ddae99b060a
---

The adapters chapter says `<seam>.backend` selects each seam's backend, oracle included; the agent catalog says `oracle.backend` routes nothing.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/02-adapters.md:28` — "`.abcd/config.json` → `<seam>.backend` selects the backend;"
- End B: `.abcd/development/brief/05-internals/01-agents.md:204` — "host-delegated, and routes nothing: host-delegated is the only value with an"

02-adapters states that `.abcd/config.json` → `<seam>.backend` selects the backend, host-delegated being only the oracle seam's default, so a different value would pick another backend. 01-agents states that `oracle.backend` records a choice and routes nothing, and the concrete oracle routes go through `oracle.roles`, `roles.<role>.runner` and `oracle.review` instead. Setting `oracle.backend` either changes routing or it does not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 778.
