---
schema_version: 1
id: "iss-2609181121522692"
slug: "no-lint-rule-checks-that-an-intent-s-bucket-agrees-with-its"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "Gropius managed-repo session gropiusllm-56, merge after a bucket-moving PR, relayed to abcd-17 on 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
---

No lint rule checks that an intent's bucket agrees with its specs' buckets, so a mis-merge that files a new planned intent's spec into closed/ (or the intent into shipped/ with its spec still open) lands silently. Observed in the Gropius managed repo on 2026-09-18 at v0.9.0: after a pull request had moved many records planned/ to shipped/ and open/ to closed/, a git merge of a branch adding one NEW planned intent and its open spec ran rename detection against those moves and offered to place the newcomers in shipped/ and closed/. The record gates read each record on its own: spec_lifecycle refuses a status key, intent_lifecycle checks ids and frontmatter placement, and neither reads the linked record's folder, so the pair "intent planned, its spec closed" passes every gate. Relayed from session gropiusllm-56, which called it a process observation rather than an abcd defect; recorded because the detector is cheap and the shape recurs wherever bucket moves and new records meet in one merge. Wanted: a record-lint rule that refuses a planned intent any of whose specs is in closed/ when no open spec remains, and a shipped intent any of whose specs is in open/. This is the inverse of itd-2609111003026787 (the gate for an intent whose work is live while its spec stays open), which reads the tree rather than the buckets, and a sibling of iss-2608290808193471; the bucket-agreement half is mechanical and needs no reading of the code.
