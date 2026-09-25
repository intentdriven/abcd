---
schema_version: 1
id: "iss-2609251053005833"
slug: "appendtoauditnotes-leaves-two-blank-lines-under-the-audit"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

appendToAuditNotes leaves two blank lines under the Audit Notes heading whenever the section already holds a block: it writes one blank after the heading and then copies the section back with its own leading blank line, trimming only the trailing ones. So the second write into a section (a condition block after the OWED stub, on every shipped record) opens it with a double blank line.
