---
schema_version: 1
id: "iss-2609251600019863"
slug: "reading-floor-a-crlf-document-whose"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "The redactor drops a CRLF pair whole when the drop runs to the end of the document, so a CRLF record whose excluded section is last redacts to its LF twin's text instead of ending in a lone carriage return the verifier refused. The verifier's lone-CR refusal is unchanged and still refuses a source that carries one."
impact: fix
resolved_by:
  commit: "a8f0af6b7da7b2b5bdb6d89c21fe7523682eb62a"
---

reading floor: a CRLF document whose excluded section is the LAST section is refused with 'ends line N with a lone carriage return': the redactor's output ends in a CR with no LF, and the verifier's lone-CR refusal then blames a CR the source does not carry. Fail-closed (no leak), but such a record can never be assembled. Fix: keep the tail's newline when the redactor drops the last section, or trim a trailing lone CR at EOF before the check; add the case to floor_lines_test's CRLF control.

## Grounds

- pursued: a CRLF document whose excluded section is first, in the middle or last redacts to its LF twin's bytes, line endings aside, with no lone CR; a lone CR in the redacted output of a source that has none, or a CRLF assembly refused for one, would show it wrong
