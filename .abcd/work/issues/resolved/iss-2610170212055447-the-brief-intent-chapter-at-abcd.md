---
schema_version: 1
id: "iss-2610170212055447"
slug: "the-brief-intent-chapter-at-abcd"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "PR7 review of the v0.15.0 deferred brief findings"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Say at those lines that a closing spec close runs the auditor on a provider route and leaves the audit owed otherwise, and that a build run audits on the closing lane."
resolution: "the intent chapter now says a shipping close makes the audit owed and runs the auditor on a provider route, and a build audits on its closing lane"
impact: internal
---

The brief intent chapter at .abcd/development/brief/04-surfaces/05-intent.md:271-272, :318 and :544 says nothing runs the intent-auditor on its own. But a spec close that ships an intent runs the auditor itself and ingests its verdict when the auditor's route reaches a provider connection (internal/surface/cli/route.go:301-306), and a build run audits on the lane whose landing closes the intent's last open spec (internal/core/implement/loop/validate.go:220-229).
