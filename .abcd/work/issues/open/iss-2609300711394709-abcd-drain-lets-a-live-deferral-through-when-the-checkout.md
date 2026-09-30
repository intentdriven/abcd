---
schema_version: 1
id: "iss-2609300711394709"
slug: "abcd-drain-lets-a-live-deferral-through-when-the-checkout"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/eligible.go"
remedy: "When an open record carries a deferral and no release tag is found, mark the anchor unknown and hand back every record carrying a deferral under the deferred rule, naming the missing tags and git fetch --tags, with the dry run saying the anchor is unknown; keep the refusal on a failed tag read; test on a --depth 1 --no-tags clone."
---

abcd drain lets a live deferral through when the checkout holds no release tag: liveDeferralAnchor returns an empty anchor when no tag is found, so every deferred_after reads as lapsed and a record a person carried past this release is eligible. A shallow or tagless clone (fetch-depth 1 fetches no tags) is the unattended drain's likely checkout, and the comment above the function promised the opposite.
