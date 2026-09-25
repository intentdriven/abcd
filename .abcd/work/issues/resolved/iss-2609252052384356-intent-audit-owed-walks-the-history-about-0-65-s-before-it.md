---
schema_version: 1
id: "iss-2609252052384356"
slug: "intent-audit-owed-walks-the-history-about-0-65-s-before-it"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_drain.go"
resolution: "The front door refuses a negative --max through intent.CheckOwedCap before the history walk; the --owed help and the page say it writes (TestIntentAuditOwedRefusesANegativeCapFirst)."
impact: internal
resolved_by:
  commit: "c2050cfa"
---

intent audit --owed walks the history (about 0.65 s) before it refuses a negative --max, and its flag help does not say it writes (it parks an OWED stub on a markerless head, a diff a peer sees)

## Grounds

- pursued: --owed --max -1 exits 2 with nothing on stderr from the history walk, and the --owed help line names that it writes; a stderr 'shipped days' line on that refusal would show it wrong
