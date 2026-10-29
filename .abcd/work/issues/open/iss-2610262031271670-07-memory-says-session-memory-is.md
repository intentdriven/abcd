---
schema_version: 1
id: "iss-2610262031271670"
slug: "07-memory-says-session-memory-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 131 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:56"
remedy: "In 07-memory.md (line 50 and the vendor-boundary paragraph) mark session_memory distillation via `dev-sync memory` as staged; 08-abcd.md and 03-configuration.md section 2 already say no dev-sync verb is registered or coded."
finding_ref: cfk-b566df961a16bdb2
---

07-memory says session memory is distilled via dev-sync memory as existing behaviour; 04-naming's register of every surface carries no dev-sync verb.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:56` — "The remaining surfaces are exempt for the plain reason that no maritime cognate"
- End B: `.abcd/development/brief/05-internals/07-memory.md:50` — "distilled by the `internal/core/memory` adapter via `dev-sync memory`"

07-memory makes the vendor memory directory a read-only harvest source and says session_memory pages are 'distilled by the `internal/core/memory` adapter via `dev-sync memory`', re-distilled on each sync. 04-naming registers every surface, maritime or exempt, so that an exemption is a decision rather than an omission, closing with 'The remaining surfaces are exempt for the plain reason that no maritime cognate' adds meaning, and no dev-sync appears anywhere in it. The memory chapter rests on a verb the naming register says abcd does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 131.
