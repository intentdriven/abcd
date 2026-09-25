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
resolution: "auto-release's detect reads the tag's newest release.yml push run and its verify job; a failed verify refuses loudly and names the re-cut (a new dated version) instead of rebuilding the refused commit, while a green verify with a failed publish, and a tag with no push run, still heal. TestAutoReleaseRefusesToRebuildATagItsVerifyRefused runs the committed step and both template renderings against a fake gh."
impact: fix
resolved_by:
  commit: "1ebd6de0e6e2620e535a504817b6141085650451"
---

auto-release's heal path rebuilds a hand-pushed tag whose verify failed on every later push to main. With the tag now made after verify on the auto-release path (iss-2608231226347380), a tag without a Release can still arise from a hand-pushed tag whose release.yml verify refused. detect cannot tell that from a transient publish failure (both are tag-present, Release-missing), so it sets need_release and rebuilds the same tagged commit, which fails again, on each push. Carried from the duplicate iss-2609100513521322's second acceptance: detect should refuse loudly and name the re-cut when the tag's last verify concluded in failure, or bound the retry.

## Grounds

- pursued: a tag its own verify refused stops the heal loop with one loud refusal and the heal survives for a transient publish failure; shown wrong by a detect that re-releases a tag whose push run's verify failed, or refuses one whose verify passed
