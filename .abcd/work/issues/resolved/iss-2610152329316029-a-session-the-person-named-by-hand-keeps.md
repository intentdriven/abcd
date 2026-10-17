---
schema_version: 1
id: "iss-2610152329316029"
slug: "a-session-the-person-named-by-hand-keeps"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "final review of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "On a clear with a title and no live entry on this host, carry the old id's given record to the new id (match by host through a small per-host note, or record the person's titles by host); test resume after clear of a hand-named Bob."
resolution: "A clear of a title abcd never gave records the new id as given nothing, so a later resume keeps the person's name; TestAClearOfAPersonsNameIsNeverAFork and TestAClearBeforeAbcdGaveTheNameStillRecordsItAsThePersons hold it."
impact: internal
resolved_by:
  intent: "itd-2610111707085098"
  spec: "spc-2610152057143798"
---

A session the person named by hand keeps its empty given record under its old id when it is cleared: the record moves only through a live entry, and a person-named session has none. A later resume of the new id, while abcd's session of that persona name runs elsewhere, reads the person's title as inherited and replaces it (C4).
