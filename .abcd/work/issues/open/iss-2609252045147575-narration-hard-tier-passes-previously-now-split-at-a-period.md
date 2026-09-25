---
schema_version: 1
id: "iss-2609252045147575"
slug: "narration-hard-tier-passes-previously-now-split-at-a-period"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
---

The launch gate suite's change-narration hard tier passes itd-65 AC3's canonical form when it is split at a sentence end: 'Previously, the ledger was a flat file. Now it is a folder.' carries 0 findings, because the previously/now construct reads one sentence at a time and needs both words in it. Remedy: when a sentence opens with 'now' and the sentence before it in the same paragraph opens with 'previously', refuse the pair as 'previously … now', located at its first sentence.
