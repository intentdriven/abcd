---
schema_version: 1
id: "iss-2609252052381777"
slug: "intent-audit-owed-labels-every-entry-shipped-not-yet"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_drain.go"
resolution: "Queued reviews carry shipped_state (dated | uncommitted | unknown); an unreadable history labels every entry 'shipped day unknown' (TestOwedQueueTellsUnknownFromUncommitted, TestIntentAuditOwedUnreadableHistoryIsUnknown)."
impact: internal
resolved_by:
  commit: "4772a4ff"
---

intent audit --owed labels every entry 'shipped (not yet committed)' and omits shipped from the JSON when the history walk fails, so a day that is unknown reads exactly like an intent that is uncommitted

## Grounds

- pursued: with HEAD on a missing branch every entry reads 'shipped day unknown' and shipped_state unknown, while a readable history still says 'not yet committed' for a working-tree intent; either label appearing for the other fact would show it wrong
