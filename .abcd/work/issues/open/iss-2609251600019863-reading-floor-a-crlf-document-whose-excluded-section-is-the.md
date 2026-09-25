---
schema_version: 1
id: "iss-2609251600019863"
slug: "reading-floor-a-crlf-document-whose-excluded-section-is-the"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
---

reading floor: a CRLF document whose excluded section is the LAST section is refused with 'ends line N with a lone carriage return': the redactor's output ends in a CR with no LF, and the verifier's lone-CR refusal then blames a CR the source does not carry. Fail-closed (no leak), but such a record can never be assembled. Fix: keep the tail's newline when the redactor drops the last section, or trim a trailing lone CR at EOF before the check; add the case to floor_lines_test's CRLF control.
