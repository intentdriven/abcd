---
schema_version: 1
id: "iss-2611041016350835"
slug: "the-release-workflow-s-verify-job-ran-go"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
remedy: "Give the Test step its own -timeout in release.yml and its scaffold template, and raise the job's timeout-minutes so every step's limit still fires first."
resolution: "The Test step now sets -timeout 20m in release.yml and its scaffold template, and the job's limit is 75 minutes, so each step's own limit fires first."
impact: fix
---

The release workflow's verify job ran go test with no -timeout, so a package past go's 10-minute default failed the job with no test failing, after the merge queue had passed the same tree under a longer limit: on 2026-11-04 internal/core/implement/loop stopped the v0.16.1 release at 600 seconds, before the tag. The scaffolded release workflow carries the same step.
