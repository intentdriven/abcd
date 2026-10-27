---
schema_version: 1
id: "iss-2610262032328073"
slug: "implement-s-record-verb-captures"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 282 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:427"
remedy: "none (filed automatically)"
finding_ref: cfk-02ad13ba10da62fe
---

Implement's record verb captures transcripts into the history store, which ahoy says holds no transcripts

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:427` — "the REGISTRY only: identity and lineage keyed on the root-commit SHA. ahoy owns it; it holds no transcripts"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:419` — "on a complete run captures the run's transcripts into the history store, one capture per path"

27-implement says the record verb, on a complete run, captures the run's transcripts into the history store, one capture per path. 01-ahoy defines the history store as the registry and says it holds no transcripts. Implement itself lists the transcript and history stores as separate stores, so the same name points at the registry in one place and the transcript destination in another. Both documents cannot be right about where a run's transcripts go.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 282.
