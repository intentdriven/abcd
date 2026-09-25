---
schema_version: 1
id: "iss-2609252127428538"
slug: "the-intent-audit-emit-errors-carry-an-absolute-path"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_drain.go"
---

The intent audit emit errors carry an absolute path unredacted: the drain's per-entry EmitError reaches the text row and the JSON emit_error of intent audit --owed, and commands/intent.md step 2 tells the host to report that text; the single intent audit <itd-N> refusal carries the same unredacted path. Both should pass through fsutil.RedactHome, as the other home-bearing messages in the cli do.
