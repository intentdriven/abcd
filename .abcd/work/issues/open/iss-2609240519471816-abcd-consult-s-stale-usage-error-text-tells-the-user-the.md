---
schema_version: 1
id: "iss-2609240519471816"
slug: "abcd-consult-s-stale-usage-error-text-tells-the-user-the"
severity: "minor"
category: "ux"
source: "drift-detection"
found_during: "v0.10.0 release gate: brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
---

abcd consult's stale-usage error text tells the user the binary predates the consult command and to rebuild it with make build, although consult is host-delegated by design, so a user rebuilds for nothing. Found by the v0.10.0 brief-surface crosscheck at fa744b41 (finding x-043).
