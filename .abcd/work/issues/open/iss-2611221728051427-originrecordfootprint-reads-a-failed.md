---
schema_version: 1
id: "iss-2611221728051427"
slug: "originrecordfootprint-reads-a-failed"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Return the cause apart from held: name a failed listing, a record held in more than one file and a refused record each in the refusal, keeping the refusal itself."
---

originRecordFootprint reads a failed listing of origin's ledger, a record origin holds in more than one file and a record the reader refuses all as not held, so the build's refusal says origin holds no such record when the truth is that it could not be read or is held twice
