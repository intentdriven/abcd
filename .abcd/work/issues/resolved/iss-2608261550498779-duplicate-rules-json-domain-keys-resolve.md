---
schema_version: 1
id: "iss-2608261550498779"
slug: "duplicate-rules-json-domain-keys-resolve"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/core/rules/rules.go"
resolution: "Load refuses a duplicate .abcd/rules.json key at any object level, mirroring capture's dup-key scan"
impact: internal
resolved_by:
  commit: "ee9e48cb"
---

Duplicate domain keys in .abcd/rules.json resolve silently last-wins: the JSON decoder keeps the final occurrence, so a repo carrying two blocks for the same domain loses the first with no diagnostic — an easy state to reach after a merge. Detect duplicate keys at load with a token-level scan and refuse or warn.