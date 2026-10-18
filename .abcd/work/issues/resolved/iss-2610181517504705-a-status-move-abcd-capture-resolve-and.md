---
schema_version: 1
id: "iss-2610181517504705"
slug: "a-status-move-abcd-capture-resolve-and"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "building the store-evidence path for iss-2610171250466572"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/ledgerroot.go"
remedy: "Carry the source record's mode to its new path on every status move, and pin it with a test that moves a 0600 record."
resolution: "a status move writes the record at its new path with the mode the record had at its old one, through fsutil.WriteFileAtomicModeOfInRoot"
impact: fix
---

A status move (abcd capture resolve, and the other moves between status folders) writes the record at its new path with the atomic writer's default mode 0644, not the mode the record had, so a record kept 0600 in the embargo store comes out readable by group and other, and every embargo sub-verb then refuses the store until the person runs chmod 600 on it. The store's folder is required to be 0700, so no other account can reach the file; the cost is the refusal and the manual step.
