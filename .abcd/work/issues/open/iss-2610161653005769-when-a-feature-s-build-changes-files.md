---
schema_version: 1
id: "iss-2610161653005769"
slug: "when-a-feature-s-build-changes-files"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "At a build's implement stage, list the open issues whose found_at falls inside the lane's footprint and that the drain rule would take (needs no decision). Fix them in the same lane as a separate commit per issue, each with its own Resolves line, reviewed by the same pair, under the same preflight and pull request. Report the ones left out and why. Keep it opt-in per build at first, cap the count so a feature PR stays reviewable, and never take an issue that needs a decision or that another lane holds. Builds on iss-2610161652341096 (bundling a drain's issues by shared files)."
blocked_by: [iss-2610161652341096]
---

When a feature's build changes files that open issues also name, those issues stay open until a separate drain lane gets to them, even though the build has the files open, reviewed and tested at that moment. The product thinker suggested on 2026-10-16 that a feature's build trigger the drain on its own files, so that every eligible issue in the code the feature touches is fixed while the feature is being built and tested.
