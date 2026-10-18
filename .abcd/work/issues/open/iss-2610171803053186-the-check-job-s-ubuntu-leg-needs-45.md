---
schema_version: 1
id: "iss-2610171803053186"
slug: "the-check-job-s-ubuntu-leg-needs-45"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Split or speed up the slowest race packages (internal/core/implement/loop first) until the ubuntu leg runs well inside 30 minutes, then set its ceiling back to 30 in ci.yml and racelanebudget_test.go."
---

The check job's ubuntu leg needs 45 minutes since the drain bundle grew the implement loop package's race run past the 30-minute ceiling it had kept; the workflow names a faster suite as the remedy for a leg near its cap.
