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
resolution: "appendToAuditNotes trims the section's leading blank lines as it trims the trailing ones, so the section opens with one blank line."
impact: fix
resolved_by:
  commit: "d12638d54ae484f8938558b93a9e4ac4e70f7baf"
---

appendToAuditNotes leaves two blank lines under the Audit Notes heading whenever the section already holds a block: it writes one blank after the heading and then copies the section back with its own leading blank line, trimming only the trailing ones. So the second write into a section (a condition block after the OWED stub, on every shipped record) opens it with a double blank line.

## Grounds

- pursued: a write into a section already holding a block leaves one blank line under the heading; TestAppendToAuditNotesOpensWithOneBlankLine would fail on a second
