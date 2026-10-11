---
schema_version: 1
id: "iss-119"
slug: "assisted-by-trailer-declared-but"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "post-merge retrospective"
found_at: "AGENTS.md"
resolution: "attribution enforcement lives in scripts/check-attribution.sh plus a CI workflow; this capture predates that gate"
impact: internal
---

AGENTS.md declares the Assisted-by trailer for AI-assisted commits, but nothing enforces it: the repo commits only pre-commit and pre-push hooks, and the global hooks are pure dispatchers. The ad3fc38 merge commit landed on a PR branch without the trailer and no gate objected. Per one-writer-per-file's sibling lesson (a convention without an armed check is silently violated under load), either a commit-msg check validates the trailer on AI-assisted commits or the miss rate is accepted explicitly.