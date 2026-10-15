---
schema_version: 1
id: "iss-2610150656007932"
slug: "the-brief-s-embark-criterion-says-the"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md"
remedy: "Amend the criterion to name brief/01-product/01-press-release.md, as the mapping and the code do; the criterion is the product thinker's to change."
---

The brief's embark criterion says the amended press release becomes the new repository's .abcd/development/brief/README.md, but the brief-to-lifeboat mapping (internal/core/lifeboat/mapping.go) and this repository keep the press release at .abcd/development/brief/01-product/01-press-release.md, the path the project site reads its brief opening from. The criterion names a file nothing writes.
