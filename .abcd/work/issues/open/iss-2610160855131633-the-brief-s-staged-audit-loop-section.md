---
schema_version: 1
id: "iss-2610160855131633"
slug: "the-brief-s-staged-audit-loop-section"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "v0.15.0 deferred findings, PR2, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Rewrite the staged audit-loop section and the configuration keys to describe the loop as abcd build's last stage per itd-50's 2026-09-21 re-scope, or drop the per-intent keys from the brief; check against itd-50 and its spec."
---

The brief's staged audit-loop section (05-intent.md, the per-intent audit_mode and audit_budget keys) and 05-internals/03-configuration.md:353-363 still describe the audit loop as a per-intent mode, while itd-50 records that on 2026-09-21 the loop was re-scoped to the last stage of abcd build, not a per-intent mode; the staged text contradicts the intent it cites. Found while fixing iss-2610160608394392, 2026-10-16.
