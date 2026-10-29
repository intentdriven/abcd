---
schema_version: 1
id: "iss-2610262035589664"
slug: "the-verification-matrix-says-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 613 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:35"
remedy: "Scope the Oracle adapter degrade row to the steps that hand a written request to the host (intent audit and its kin), and add that the reading ingest's dispatch form exits 2 with nothing ingested when its provider cannot be reached."
finding_ref: cfk-4995de26632bb120
---

The verification matrix says an unreachable provider falls back to the harness rather than failing; the reading chapter's dispatch form exits 2 with nothing ingested when the provider cannot be reached

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:35` — "A step routed to a provider connection that cannot be reached, nothing having been sent, runs through the harness rather than failing the run"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:377` — "a provider that could not be reached, a run id outside the run-id grammar, a run no assembly parked, and a dispatch that also names an output each exit 2 with nothing ingested"

The matrix's oracle-adapter-degrade row says a step routed to a provider connection that cannot be reached, nothing having been sent, runs through the harness rather than failing the run, and the receipt names the harness used. The reading chapter says that for its dispatch form a provider that could not be reached exits 2 with nothing ingested. It also refuses an output the host produced while the position is routed to a provider, so the step cannot fall back to the harness.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 613.
