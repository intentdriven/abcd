---
schema_version: 1
id: "iss-2611221756090856"
slug: "the-grounds-package-s-comments-describe"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "mining the parked phase-9 clone, 2026-11-22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/grounds/grounds.go"
remedy: "Reword both comments to name the Grounds bullet as the one carrier written, and the frontmatter scalar as a legacy shape that is tolerated on read and refused by the record lint."
---

The grounds package's comments describe a carrier the record retired. Parse's doc comment in internal/core/grounds/grounds.go calls the token grammar the one spelling written to a `grounds:` frontmatter scalar and to a bullet alike, and the comment on spaceRunRe names a YAML scalar in the ledger as one of the two carriers. No write path produces a frontmatter `grounds:` any more: the schema tolerates the key as a legacy one whose value nothing reads, and the record lint refuses it and names the Grounds section. A reader of the package learns of a writer the same tree refuses. A detection reading of 2026-09-02 found it, and the answer given to that item accepted it and promised the comment a fix that no record then carried.
