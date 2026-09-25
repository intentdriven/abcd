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
resolution: "The narration gate reads a pair across a sentence end: a sentence opening with 'now' after one opening with 'previously' in the same run of text is refused as 'previously … now', located at the first sentence and named whole; two split-form refusals and one qualifying-'previously' pass are in narrationSpecification."
impact: fix
resolved_by:
  commit: "dcd5e124"
---

The launch gate suite's change-narration hard tier passes itd-65 AC3's canonical form when it is split at a sentence end: 'Previously, the ledger was a flat file. Now it is a folder.' carries 0 findings, because the previously/now construct reads one sentence at a time and needs both words in it. Remedy: when a sentence opens with 'now' and the sentence before it in the same paragraph opens with 'previously', refuse the pair as 'previously … now', located at its first sentence.

## Grounds

- pursued: 'Previously, the ledger was a flat file. Now it is a folder.' and 'Previously the gate read JSON. Now it reads YAML.' hard-fail with the file, line and both sentences named, 'Run the previously saved query. Now run the gate.' passes, and the survey over every Markdown file adds no location; either split form passing, the qualifying pass refused, or a new survey location would show it wrong
