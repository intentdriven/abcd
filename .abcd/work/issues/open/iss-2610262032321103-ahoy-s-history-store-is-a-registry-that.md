---
schema_version: 1
id: "iss-2610262032321103"
slug: "ahoy-s-history-store-is-a-registry-that"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 281 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:427"
remedy: "none (filed automatically)"
finding_ref: cfk-7181fa116679321c
---

Ahoy's history store is a registry that holds no transcripts; build's history store is where a runner's transcript lands

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:427` — "the REGISTRY only: identity and lineage keyed on the root-commit SHA. ahoy owns it; it holds no transcripts"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:499` — "the runner's transcript lands in abcd's history store, keyed on the repository's root commit"

01-ahoy's user-scope inventory defines history/ as the registry only, keyed on the root-commit SHA, and says it holds no transcripts, with the transcript corpus kept in the sibling transcripts/< root-sha>/. 34-build says a runner's transcript lands in abcd's history store, keyed on the repository's root commit. So one name covers two stores: a reader of build cannot tell which directory a lane's transcript goes to, and a reader of ahoy is told the history store never holds one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 281.
