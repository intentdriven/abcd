---
schema_version: 1
id: "iss-2608261551077971"
slug: "rule-injection-has-no-per-repo-size"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/core/rules/rules.go"
resolution: "injection is bounded by a 64KiB per-repo budget with a loud truncation notice; truncated domains retry, never silently drop"
impact: fix
resolved_by:
  commit: "ee9e48cb"
---

Rule injection has no per-repo size budget: recall keywords and rule bodies are repo-controlled, so a hostile or bloated rules.json can push roughly 260KB into a session's context in one turn (recurring, not once: the refresh backstop clears the dedup ledger every 15 prompts by default and it is also cleared on every SessionStart/PreCompact, so the payload re-lands at least every 15 turns and after each compaction). The existing 256KiB read cap bounds the file, not the injected total across domains. Decide and enforce a per-repo injection budget with loud truncation.