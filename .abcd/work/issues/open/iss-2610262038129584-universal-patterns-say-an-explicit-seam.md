---
schema_version: 1
id: "iss-2610262038129584"
slug: "universal-patterns-say-an-explicit-seam"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 779 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:181"
remedy: "none (filed automatically)"
finding_ref: cfk-9965d16dfeac77eb
---

Universal patterns say an explicit `<seam>.backend` value selects a wired backend for history, spec, run and scanner; the configuration chapter lists those selectors as staged keys no code reads.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:181` — "An explicit value selects a wired external backend (richer capability, unusual setups, or testing)."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:382` — "No shipped code reads any of the keys below."

04-universal-patterns describes backend resolution through `<seam>.backend` for the history, spec, run and scanner seams, with an explicit value selecting a wired external backend. 03-configuration puts `spec.backend`, `run.backend` and `history.backend` under staged config keys and says no shipped code reads any of them, and nothing chooses between backends. Writing such a key either selects a backend or has no effect.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 779.
