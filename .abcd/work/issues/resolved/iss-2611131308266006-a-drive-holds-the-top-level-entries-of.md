---
schema_version: 1
id: "iss-2611131308266006"
slug: "a-drive-holds-the-top-level-entries-of"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/remoteid.go"
remedy: "Hold the hooks tree at every depth and the watched settings at every scope across lanes, prove an in-tree hooks copy against origin's tip at a drive's start, and state what one account cannot hold."
resolution: "A drive now holds the checkout's hooks tree at every depth, read from the hooks directory git takes at the checkout from every scope, with info/attributes and every watched setting at every scope across its lanes; the lane's pin proves its hooks at every depth and refuses a symlink that leaves the directory; a drive's start proves an in-tree hooks copy against origin's tip; the implement page states what one account cannot hold."
impact: fix
---

A drive holds the top-level entries of the checkout's hooks directory and a listed set of settings across its lanes; the implement page lists what it does not hold.
