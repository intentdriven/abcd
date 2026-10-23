---
schema_version: 1
id: "iss-2610231038335615"
slug: "reading-the-reservations-writes-into-the"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/core/claim/view.go"
origin: researcher-authored
production_mode: hand-written
remedy: "Name the empty tree without writing it on the read paths (hash-object without -w, or the known object name for the repository's hash), keep the writing form for the paths that commit a reservation, and add a test that a view leaves .git/objects unchanged."
resolution: "reading the reservations names the empty tree without writing it (hash-object without -w); the writing form stays for the paths that commit a reservation"
impact: fix
resolved_by:
  commit: "27d1120ec"
---

Reading the reservations writes into the shared git object store: claim.View names the empty tree through gitutil.EmptyTree, which runs git hash-object -w, so every read of the reservations (abcd peers, the status line, the sweep's dry run) writes or refreshes the empty-tree object under .git/objects. A read only compares tips against the empty tree's name and never needs the object to exist, and the sweep spec's criterion 11 allows writes only under the abcd home and the local tier.
