---
schema_version: 1
id: "iss-2610151617249338"
slug: "a-fix-to-a-plugin-payload-file-committed"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "v0.14.2 release run 38763688586"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
remedy: "Have launch receipts (and make preflight on a release branch) run launch archive --verify against the committed pin, so a payload edit made after the roll is refused before the release PR merges rather than at the release run's verify job; and have the runbook say a post-roll payload fix re-pins the archive in the same commit."
---

A fix to a plugin-payload file committed after the release roll changes the plugin archive's digest, and nothing before the release run notices. On 2026-10-15 the v0.14.2 cut fixed docs/how-to/install.md after the content commit (a docs-currency finding); make preflight passed, abcd launch receipts passed, the release PR merged, and only the release workflow's verify job refused: the committed marketplace pin named the archive rendered at the roll (beb4343893…), the merged tree rendered another (28755d0cf4…). No tag was cut, so the release could be repaired by re-pinning on main, but it cost a merge-queue cycle and a second release run.
