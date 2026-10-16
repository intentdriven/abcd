---
schema_version: 1
id: "iss-2610160608453195"
slug: "abcd-update-help-internal-surface-cli"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/update.go"
remedy: "Say 'release.yml on main' in the provenance sentence of update's long help in internal/surface/cli/update.go."
---

abcd update --help (internal/surface/cli/update.go) says build provenance is pinned to intentdriven/abcd's release.yml and to the tag's commit, leaving out that the signer must be release.yml on refs/heads/main, which commands/update.md, docs/how-to/install.md and internal/core/update/provenance.go require.
