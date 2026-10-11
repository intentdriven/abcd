---
schema_version: 1
id: "iss-2609261950066257"
slug: "the-bare-board-reports-its-directory-as"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSite"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/core.go"
resolution: "core.Status reports Dir through fsutil.RedactHome, so abcd --json and the text board name a checkout under the home as ~/...; the inspection still reads the absolute directory. The board has no repository root to be relative to, since the directory is what it reports."
impact: fix
resolved_by:
  commit: "c5eab8b89"
---

The bare board reports its directory as an absolute path in machine output: abcd --json carries dir as filepath.Abs of the working directory (core.Status, internal/core/core.go, embedded in the board envelope by internal/surface/cli/cli.go), so a checkout under the home names the developer in --json, against the iss-81 rule the site and lifeboat verbs are held to. The text board prints the same field on its first line.

## Grounds

- pursued: a checkout under HOME reports dir as ~/src/repo while IsGitRepo is still read from the real directory (TestStatusNamesTheDirectoryWithoutTheHome); an absolute dir in abcd --json would show it wrong
