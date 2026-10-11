---
schema_version: 1
id: "iss-116"
slug: "the-walkfiles-skip-set-covers-only-node"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "itd-95 P1 review"
found_at: "internal/core/lifeboat/probe.go"
resolution: "Widened walkSkipDirs to the common Python/build-output/CocoaPods trees (.venv, venv, .tox, __pycache__, target, build, dist, Pods); vendored markers no longer cited and the dot-prefixed tree no longer eats the cap before src/."
impact: fix
---

The WalkFiles skip set covers only Node and Go dependency trees, so a Python .venv, a Rust target/, __pycache__, dist/, build/ and Pods/ are walked as if they were the team's own source. Two consequences on a record-less repo: the open-questions adapter cites a vendored dependency's TODO as this project's open question, and because fs.WalkDir reads directories in lexical order a large dot-prefixed dependency tree can consume the 50000-file walk cap before the project's own src/ is reached. Found by an independent review of the itd-95 diff; spc-12 names the current skip set, so widening it is a design revisit rather than an implementation fix.