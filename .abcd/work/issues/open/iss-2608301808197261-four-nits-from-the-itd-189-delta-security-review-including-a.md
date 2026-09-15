---
schema_version: 1
id: "iss-2608301808197261"
slug: "four-nits-from-the-itd-189-delta-security-review-including-a"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-189-delta-security"
found_at: "internal/core/lint/schema.go"
---

four nits from the itd-189 delta security review including an unpinned bucket field stand down and a zero width space passing the gate

Four nits from the itd-189 delta security review. Per the round rule adopted
2026-08-30 these are settled at the ship commit rather than by re-opening a
review.

1. `checkRecordBucketField` now stands down on an absent bucket field and
   delegates to `checkRecordRequiredFields`. That holds only because `adm` is the
   sole store declaring a `bucketField` and `run` is in its required set. There
   is no pin equivalent to `TestEveryJoinTargetPositionIsADeclaredPosition`, so a
   future store declaring a `bucketField` outside its required set gets a silent
   absence. Same asymmetry `iss-2608301634527391` records one leg over.
2. The position leg `continue`s before the bucket leg, so an admission that is
   both cross-bucket and cross-position reports only the position, and the author
   needs two lint rounds to converge.
3. `grounds: <U+200B>` passes the gate: `strings.TrimSpace` does not treat a
   zero-width space as whitespace, though it does trim U+00A0. Same consequence
   as the spelling class, different mechanism, and the codebase already knows
   this character elsewhere.
4. `grounds: *a`, an alias to an undefined anchor, passes the gate; a strict YAML
   parser errors on it.
