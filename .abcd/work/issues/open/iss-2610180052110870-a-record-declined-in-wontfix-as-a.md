---
schema_version: 1
id: "iss-2610180052110870"
slug: "a-record-declined-in-wontfix-as-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "shipping itd-166 (issue-drift shipped_unresolved)"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/capture/drift.go"
remedy: "Let capture resolve take a wontfix record whose related_intents name the intent being shipped, moving it to resolved/ with the resolve verb's fields and dropping wontfix_reason, or have spec close do it for every such record of the intent it ships; tested by shipping an intent with a duplicate declined in its favour."
---

A record declined in wontfix/ as a duplicate of an intent cannot be moved to resolved/ when that intent ships: capture resolve refuses it as already in wontfix, yet the issue-drift gate refuses a shipped intent whose related_issues name a wontfix record, so the only way through is a hand move of the record (done for iss-2608290811463906 when itd-166 shipped).
