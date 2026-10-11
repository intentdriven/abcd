---
schema_version: 1
id: "iss-120"
slug: "local-tree-id-minting-races-under"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "post-merge retrospective"
found_at: "internal/core/capture/capture.go"
resolution: "Local-tree id minting no longer races under parallel programmes: minting is now max+1 over the working tree AND every git ref, with loud degrade when git is unreadable and the armed record-lint detectors as the residual-window backstop."
impact: fix
---

Sequential record ids are minted from the local tree, so two parallel programmes mint the same next id — both took spc-10, spc-11, iss-110 and iss-111, costing four renumber commits across the episode. This generalises iss-115 (spec ids lack a uniqueness lint): issue and intent ids have armed detectors via the shared validateIDUnique primitive, spec and ADR ids have none, and detection-after-collision is the only mitigation anywhere. The design call is whether detection everywhere is enough or minting itself should be made collision-free (id-range leases per programme, mint-at-merge), which trades away human-readable sequential ids.