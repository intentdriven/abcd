---
schema_version: 1
id: "iss-2609251125599536"
slug: "auto-release-s-heal-path-rebuilds-a-hand-pushed-tag-whose"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

auto-release's heal path rebuilds a hand-pushed tag whose verify failed on every later push to main. With the tag now made after verify on the auto-release path (iss-2608231226347380), a tag without a Release can still arise from a hand-pushed tag whose release.yml verify refused. detect cannot tell that from a transient publish failure (both are tag-present, Release-missing), so it sets need_release and rebuilds the same tagged commit, which fails again, on each push. Carried from the duplicate iss-2609100513521322's second acceptance: detect should refuse loudly and name the re-cut when the tag's last verify concluded in failure, or bound the retry.
