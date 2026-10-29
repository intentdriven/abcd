---
schema_version: 1
id: "iss-2610262033488570"
slug: "the-memory-substrate-has"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 439 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:16"
remedy: "Edit 07-memory.md section 0: mark the principle-distiller routing rule and the user-scope pages as staged (07-library 'Not built'; 03-configuration says nothing resolves ~/.abcd.noindex/memory/ today)."
finding_ref: cfk-4397f5d197a7001e
---

The memory substrate has principle-distiller route pages between a repo scope and a user scope; the library chapter says the curator role and user-scope memory are not built.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:16` — "Which scope a curated page lands in is decided by `principle-distiller` at curation time"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:413` — "Not built: a curator role on the `principle-distiller` agent (the agent and the disembark sub-verb it feeds ship; this role does not), ingest and ask run reports, user-scope memory outside the repo, and the automatic extraction of a spec's modification grammar into the store at spec completion."

07-memory says the scope a curated page lands in is decided by principle-distiller at curation time, across a repo scope and a user scope at ~/.abcd.noindex/memory/, and describes retrieval querying both. 07-library lists as not built 'a curator role on the principle-distiller agent', 'user-scope memory outside the repo', and the modification-grammar extraction 07-memory also describes. A routing decision made by a role that does not exist, into a scope that does not exist, cannot be current behaviour.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 439.
