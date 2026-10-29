---
schema_version: 1
id: "iss-2610262030576576"
slug: "invariant-4-says-all-inter-agent-data-is"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 67 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:15"
remedy: "In 03-invariants.md invariant 4 (and 05-internals/04-universal-patterns.md section 4), scope 'all inter-agent data is JSON' to data the core validates and name the exceptions: a Markdown artefact built as a model's context (history reconstruct) and the prose-verdict reviewers (see iss-2610262031138048)."
finding_ref: cfk-97d08d265a8dd54b
---

Invariant 4 says all inter-agent data is JSON and markdown is only a render step; the history chapter's reconstruct artefact is Markdown precisely because a model reads it.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:15` — "all inter-agent data is JSON; markdown is a render step"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:149` — "The artefact is Markdown because its consumer is a model being handed the session as context"

03-invariants says all inter-agent data is JSON and markdown is a render step. 11-history says reconstruct writes one self-contained Markdown artefact as its primary output, and that it is Markdown because its consumer is a model being handed the session as context. Only the telemetry file beside it is JSON. A Markdown document designed as a model's input is inter-agent data that is not JSON, so the invariant's 'all' does not hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 67.
