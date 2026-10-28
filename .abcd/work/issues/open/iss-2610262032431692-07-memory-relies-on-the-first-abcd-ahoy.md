---
schema_version: 1
id: "iss-2610262032431692"
slug: "07-memory-relies-on-the-first-abcd-ahoy"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 308 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:42"
remedy: "none (filed automatically)"
finding_ref: cfk-e6b3b1048d8f53f6
---

07-memory relies on the first /abcd:ahoy after itd-36 backfilling memory pages' source class; ahoy's detection and apply passes carry no such step

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:42` — "backfilled with `source.class: session_memory` as the default on first `/abcd:ahoy` after itd-36 ships"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:591` — "What it probes, in behaviour rather than in step order: the folder's kind and the plugin root"

07-memory makes ahoy the step that backfills source.class: session_memory onto pre-existing memory pages. 01-ahoy enumerates everything its one detection pass probes and every gap category the apply pass acts on, and an install with no detected gap writes nothing; none of those names the memory store or a source-class backfill. As written, ahoy cannot perform the step 07-memory depends on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 308.
