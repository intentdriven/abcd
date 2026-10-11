---
schema_version: 1
id: "iss-152"
slug: "brief-surface-crosscheck-residual-non"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "v0.4.2-era drift snapshot superseded by the v0.6.2 systematic brief-surface crosscheck pass; the live tracker is the drift intent line, not per-release snapshots"
impact: internal
---

brief-surface crosscheck residual (non-deterministic tail after 4 tier-full rounds, 27 items fixed): (1) 04-surfaces/01-ahoy.md bare-render enumeration omits the guard: health line the shipped bare 'abcd ahoy' now prints (5 lines not 4); (2) 04-surfaces/07-memory.md:104 enumerates 4 launch dry-run gates but marker-block and documentation-auditor are Status:not_implemented (Phase-5 deferred) stubs — only secret/PII scan + installability smoke actually run. Both minor brief-internal staleness, no user-facing doc affected (docs-currency PROMOTE). Deferred from v0.4.1 release gate via disposition; fix in a brief-currency sweep.