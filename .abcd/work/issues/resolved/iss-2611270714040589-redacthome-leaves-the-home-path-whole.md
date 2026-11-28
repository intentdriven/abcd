---
schema_version: 1
id: "iss-2611270714040589"
slug: "redacthome-leaves-the-home-path-whole"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
remedy: "Match the cleaned spelling of HOME as well as the raw one, and test both"
resolution: "RedactHome and its siblings now match HOME as written, cleaned and symlink-resolved in one longest-first pass, and redact the person's home inside a stage"
impact: fix
---

RedactHome leaves the home path whole when HOME is spelled with a doubled slash or a dot segment
