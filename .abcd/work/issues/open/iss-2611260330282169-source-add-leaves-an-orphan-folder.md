---
schema_version: 1
id: "iss-2611260330282169"
slug: "source-add-leaves-an-orphan-folder"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/source/add.go"
remedy: "stage the whole add in a temp folder beside the corpus and move it into place as one rename, so a key's folder appears only complete; until then, name the orphan folder in the refusal so it can be cleared by hand"
---

source.Add is not transactional: a process killed part-way through an add leaves the key's folder in the corpus with a partial or missing document and entry, and that orphan folder refuses every later add of the same key
