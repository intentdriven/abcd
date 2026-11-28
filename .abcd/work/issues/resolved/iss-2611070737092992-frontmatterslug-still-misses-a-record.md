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
resolution: "the slug-cap reader reads a record's slug field through frontmatter.Fields, the scanner record-lint uses, so a byte-order mark, a spaced key and a leading comment no longer hide the field from the commit-message hook or the record_slug_cap rule, and a trailing comment is no longer counted into the slug's length; it reads the record through the capped record read, so a record over the 1 MiB record cap, the only kind with a line over 1 MiB, is named by the rule rather than read, and an unclosed block carries no slug field, as record-lint reads it"
impact: fix
---

frontmatterSlug still misses a record that starts with a byte-order mark, a line written 'slug : value', and a slug field that follows a line over 1 MiB. The commit-message hook alone can miss these; record-lint still refuses them through the filename and the fields.
