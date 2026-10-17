---
schema_version: 1
id: "iss-2610170833197571"
slug: "when-origin-s-root-gitattributes-cannot"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "independent review of drain step 11, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/generated.go"
remedy: "Warn about an unreadable declaration only when the file could declare linguist-generated (its bytes name the attribute, or a macro could), and otherwise read it as declaring none in silence"
---

When origin's root .gitattributes cannot be read as a declaration (over 256 KiB, holding a NUL byte, or not a regular file), every reserve warns that generated files were not recognised, even when that file declares no linguist-generated at all, so a repository that never declared any generated file gets a warning on every reserve
