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
---

Every reservation view with a live footprint re-reads and re-matches origin's whole tree against the root .gitattributes (the claim package's generated-file reader), about 0.3 s on a large tree (150,000 files) and at most about half a second by the matching budget, and nothing keeps the answer between views although it can change only when origin's default branch moves
