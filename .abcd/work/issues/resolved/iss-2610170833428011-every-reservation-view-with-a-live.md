---
schema_version: 1
id: "iss-2610170833428011"
slug: "every-reservation-view-with-a-live"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "independent review of drain step 11, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/generated.go"
remedy: "Keep the generated set keyed on the default branch's tree id in the local tier, guarded and size-capped, and reuse it while that id is unchanged, so a view pays the match once per default-branch move"
resolution: "The generated set is kept in the checkout's local tier under the tree origin's default branch names, the matching budgets and the running binary, and read from that one tree; a later view on the same tree takes it without listing or matching again, so a view pays the match once per move of the default branch. A kept file that git tracks, is a symbolic link, is past 16 MiB or answers another key is read afresh. Tests count the matches through a seam: a second read and a second view match once, a new tree or budget matches again, and each kind of bad kept file is read afresh."
impact: fix
resolved_by:
  commit: "482ac1dc19b73a05f4764b1dac50265e2d5ae224"
---

Every reservation view with a live footprint re-reads and re-matches origin's whole tree against the root .gitattributes (the claim package's generated-file reader), about 0.3 s on a large tree (150,000 files) and at most about half a second by the matching budget, and nothing keeps the answer between views although it can change only when origin's default branch moves
