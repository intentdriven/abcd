---
schema_version: 1
id: "iss-2611221728031097"
slug: "the-comments-on-the-re-run-s-network"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/matrix/rerun.go"
remedy: "Name the unpinned field names in both rerun.go comments (the rerun struct's network field and the screens method), in the words run.go's own comments on the run record's screen already use."
---

the comments on the re-run's network copy in rerun.go say only a bare version and the record's hex digests are emptied before the network screen, but the run record's copy also has the unpinned field names abcd builds emptied (run.go, reached from the re-run), so two comments understate what is withheld
