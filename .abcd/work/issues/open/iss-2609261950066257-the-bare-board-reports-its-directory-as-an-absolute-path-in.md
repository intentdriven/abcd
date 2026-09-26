---
schema_version: 1
id: "iss-2609261950066257"
slug: "the-bare-board-reports-its-directory-as-an-absolute-path-in"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSite"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/core.go"
---

The bare board reports its directory as an absolute path in machine output: abcd --json carries dir as filepath.Abs of the working directory (core.Status, internal/core/core.go, embedded in the board envelope by internal/surface/cli/cli.go), so a checkout under the home names the developer in --json, against the iss-81 rule the site and lifeboat verbs are held to. The text board prints the same field on its first line.
