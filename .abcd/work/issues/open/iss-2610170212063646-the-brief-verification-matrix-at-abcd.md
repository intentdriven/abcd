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
remedy: "Replace the deferral at :56 with the two places the auditor runs without a person: a closing spec close on a provider route, and a build run's closing lane."
---

The brief verification matrix at .abcd/development/brief/06-delivery/02-verification-matrix.md:56 says running the intent-auditor automatically is deferred. A spec close that ships an intent runs the auditor itself when the auditor's route reaches a provider connection (internal/surface/cli/route.go:301-306), and a build run audits on the lane whose landing closes the intent's last open spec (internal/core/implement/loop/validate.go:220-229).
