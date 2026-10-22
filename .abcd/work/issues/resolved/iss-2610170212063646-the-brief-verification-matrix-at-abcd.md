---
schema_version: 1
id: "iss-2610170212063646"
slug: "the-brief-verification-matrix-at-abcd"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "PR7 review of the v0.15.0 deferred brief findings"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Rewrite the row at :56: a closing spec close makes the audit owed (receipt, marker and request), runs the auditor on a provider route, and a build run audits on its closing lane; drop the nothing-is-enqueued and deferral clauses."
resolution: "the verification matrix's lifecycle-hook row now says a shipping close makes the audit owed and runs the auditor on a provider route, and a build audits on its closing lane"
impact: internal
---

The brief verification matrix at .abcd/development/brief/06-delivery/02-verification-matrix.md:56 says running the intent-auditor automatically is deferred. A spec close that ships an intent runs the auditor itself when the auditor's route reaches a provider connection (internal/surface/cli/route.go:301-306), and a build run audits on the lane whose landing closes the intent's last open spec (internal/core/implement/loop/validate.go:220-229).
