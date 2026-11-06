---
schema_version: 1
id: "iss-2611060603321598"
slug: "a-surface-under-a-planned-intent-gets-no"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog"
remedy: "Let a cut's payload carry a line citing a planned intent marked as a partial delivery, with the ingest checking the intent's spec names the surface, and the intent staying planned."
---

The ingest's bijection lets a changelog line cite only records in the cut, so a surface that ships under a still-planned intent has no line of its own. abcd drain --drive shipped in v0.16.2 with its intent planned, and could only be introduced inside the line of the --until issue.
