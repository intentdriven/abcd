---
schema_version: 1
id: "iss-2610110348151970"
slug: "lint-unwrapprinciplelinks-ends-a-link"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/principles.go"
remedy: "unwrap principle links through the reading floor's CommonMark inline link scanner, moved to mdrecord so lint and reading share it"
resolution: "Principle statement links now unwrap through the shared CommonMark inline link scanner (mdrecord.UnwrapLinkPass), so a destination with parentheses, an escaped paren, an angle destination or a title is read whole."
impact: fix
resolved_by:
  commit: "ac7bff0e7"
---

lint.UnwrapPrincipleLinks ends a link destination at its first ')' and a label at its first ']', so a principle statement linking [label](https://x/Audit_(finance)) projects a fragment of the address into the reading bundle
