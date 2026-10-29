---
schema_version: 1
id: "iss-2610262031110865"
slug: "the-dependencies-page-presents-the-run"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 97 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:43"
remedy: "In the Adapters over native defaults list of 02-dependencies.md, mark the spec (ccpm), run (Claude Workflows, harness loop, native loop) and history (specstory) adapters as design targets, matching 05-internals/02-adapters.md ('only scanner is a seam directory'), and say no registry selects a backend yet."
finding_ref: cfk-ba721bcc7ec553f4
---

The dependencies page presents the run seam (and the spec and history seams) as live, with external loops behind one contract, but the adapters chapter says those seams are design targets

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:43` — "Claude Workflows, the companion harness's agent loop, or a thin native Go loop behind one contract."
- End B: `.abcd/development/brief/05-internals/02-adapters.md:34` — "are design targets, each introduced by the first intent that consumes it"

The dependencies page says every capability abcd once delegated to a bundled tool has a native default with the external tool available as an opt-in adapter over the same seam. As live examples it gives the run seam, with Claude Workflows, the companion harness's loop or the native loop behind one contract, the spec seam with ccpm, and the history seam with specstory. The adapters chapter says only `scanner` exists as a seam. It calls `oracle`, `history`, `spec` and `run` design targets, each to be introduced by the first intent that consumes it, and says no registry for choosing a backend exists. External loops cannot be available behind a seam that is not built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 97.
