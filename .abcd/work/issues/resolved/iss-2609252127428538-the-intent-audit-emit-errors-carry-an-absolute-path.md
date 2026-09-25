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
resolution: "The drain's emit_error (text row and JSON) and the single intent audit refusal pass through fsutil.RedactHome."
impact: fix
resolved_by:
  commit: "423b2f6d"
---

The intent audit emit errors carry an absolute path unredacted: the drain's per-entry EmitError reaches the text row and the JSON emit_error of intent audit --owed, and commands/intent.md step 2 tells the host to report that text; the single intent audit <itd-N> refusal carries the same unredacted path. Both should pass through fsutil.RedactHome, as the other home-bearing messages in the cli do.

## Grounds

- pursued: an emit error whose path lies under HOME renders as ~/... in the text row, the JSON emit_error and the single audit refusal (TestIntentAuditEmitErrorsRedactHome); the absolute path in any of the three would show it wrong.
