---
schema_version: 1
id: "iss-2610150749065106"
slug: "the-index-generator-reads-a-register"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "iss-2608220150157511 fix review pair"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/indexgen.go"
remedy: "Preserve the file's own line endings when writing a regenerated region, with a test on a CRLF document."
---

The index generator reads a register with CRLF line endings normalised to LF and writes the whole file back as LF, touching bytes outside the generated markers; the lint step normalises too, so the churn is silent. No register is CRLF today.
