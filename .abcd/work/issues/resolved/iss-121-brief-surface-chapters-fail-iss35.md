---
schema_version: 1
id: "iss-121"
slug: "brief-surface-chapters-fail-iss35"
severity: "major"
impact: internal
category: "observation"
source: "agent-finding"
found_during: "v0.4.0 release gate"
found_at: ".abcd/development/brief/04-surfaces/"
resolution: "The brief's 17 surface docs are reconciled with the shipped binary: all 95 adversarially confirmed discrepancies fixed (68 rewrites, 14 surface documentations, 8 deletions of abandoned-design prose, 5 staged markings), plus the orphaned cross-references and the stale superseded banner the deletions exposed. 7 of the original 102 findings were refuted in triage (duplicates, wrong-reality, legitimately staged)."
---

The brief's surface chapters fail the iss35 brief-surface crosscheck at full depth: 102 unique discrepancies across the 16 surface chapters plus the skills internals page (54 false-claim, 16 undocumented-surface, 13 fictional-layout, 13 stale-count, 6 criterion-violation). Examples: 02-disembark.md describes a dev-sync verb, an agent-dispatch pack flow, and a backgrounded 'disembark to' sub-verb, none of which the binary ships; 01-ahoy.md omits seven shipped install flags and the identity-pin write. The v0.4.0 release is blocked fail-closed at receipt_gate until the record is reconciled and the crosscheck PROMOTEs. Full findings preserved in the local scratch tier (2026-07-22-iss35-crosscheck files).