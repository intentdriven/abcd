---
id: dcn-2607130000000005
slug: audit-engine-uses-severity-vocabulary
date: 2026-07-13
---
2026-07-13 (itd-85 M2): audit engine uses severity vocabulary error|warn|off
(repolinter/Conftest), NOT the record-lint engine's blocker|warn, because it
maps directly onto the tri-state exit code (error->2, warn->1) and reads right
in a human render. Reused docs-lint findings (blocker|warn) get mapped to
error|warn at the docs-currency rule boundary in M3, not in the engine.
