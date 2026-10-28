---
schema_version: 1
id: "iss-2610262031186949"
slug: "02-dependencies-includes-a-thin-native"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 112 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:44"
remedy: "none (filed automatically)"
finding_ref: cfk-9145786f18dc419e
---

02-dependencies includes a thin native Go loop among the run seam's loops; the loop glossary entry says abcd never owns the run loop, an adapter supplying it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:44` — "or a thin native Go loop behind one"
- End B: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop."

02-dependencies lists the run seam's loops as host workflows, the companion harness's agent loop 'or a thin native Go loop behind one' contract, the native loop being the fallback. The glossary's loop entry says 'abcd never owns the run loop.': abcd defines the contract and a pluggable adapter supplies the loop, which it gives as the whole reason the run-loop sense needs its own name. A native Go loop shipped in abcd is a run loop abcd owns, so the two cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 112.
