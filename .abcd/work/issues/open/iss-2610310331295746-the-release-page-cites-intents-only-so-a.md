---
schema_version: 1
id: "iss-2610310331295746"
slug: "the-release-page-cites-intents-only-so-a"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release/page.go"
remedy: "Let the release page carry a breaking change from any record in the cut, or have the page name the changelog's breaking section whenever it is not empty."
---

The release page cites intents only, so a breaking change that comes from an issue, such as the sites no longer publishing author names in v0.16.0, appears in the changelog's breaking section but never on the release page a reader starts from.
