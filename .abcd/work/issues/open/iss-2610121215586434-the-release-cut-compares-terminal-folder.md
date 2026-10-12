---
schema_version: 1
id: "iss-2610121215586434"
slug: "the-release-cut-compares-terminal-folder"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "release cut dry-run v0.13.4..origin/main, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog/shipped.go"
remedy: "Compute Added and Removed by record id: an id present at both refs is in neither, whatever its slug. Keep shipped_in filtering on Added. Replace TestShippedSinceRenameReadsAsDeleteAndAdd with a test that a same-folder re-slug is absent from both sides."
---

The release cut compares terminal-folder record PATHS between the anchor tag and HEAD (internal/core/changelog/shipped.go:286-303), so the 40-character slug rename (255463f0d, PR #904) makes 1724 records read as removed and re-added: the v0.13.4..origin/main dry-run reports 1770 added and 1724 removed against 73 records that actually entered resolved/ or shipped/. The changelog check then requires about 1797 ids to be cited, and the doc-fidelity population lists about 78 intents instead of 4.
