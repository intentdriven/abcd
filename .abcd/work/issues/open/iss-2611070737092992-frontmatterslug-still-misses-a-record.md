---
schema_version: 1
id: "iss-2611070737092992"
slug: "frontmatterslug-still-misses-a-record"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recordid/slugcap.go"
remedy: "Read the walk through frontmatter.Fields instead of the hand scan, so the hook and record-lint agree, and add a test per case."
---

frontmatterSlug still misses a record that starts with a byte-order mark, a line written 'slug : value', and a slug field that follows a line over 1 MiB. The commit-message hook alone can miss these; record-lint still refuses them through the filename and the fields.
