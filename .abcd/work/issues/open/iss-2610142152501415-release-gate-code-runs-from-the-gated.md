---
schema_version: 1
id: "iss-2610142152501415"
slug: "release-gate-code-runs-from-the-gated"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "deferral discipline review (feat/deferral-discipline, third read)"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
remedy: "Build the gate binary from the base, the default branch before the merge or the previous release tag, rather than from the gated tree, or require a code-owner review on internal/core/lint and cmd/record-lint."
---

release.yml's verify job runs go run ./cmd/record-lint from the gated tree, so a release pull request that edits internal/core/lint or cmd/record-lint disarms the receipt gate as surely as editing release.yml, with only pull-request review in the way. The same holds for every gate the verify job runs from the tree it gates; it predates the deferral discipline.
