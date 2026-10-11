---
schema_version: 1
id: "iss-254"
slug: "hooks-single-provisioning-point-and"
severity: "major"
category: "bug"
source: "manual-test"
found_during: "iss-253 diagnosis, 2026-08-16"
found_at: "hooks/hooks.json"
resolution: "Every binary-invoking hook (UserPromptSubmit, PreToolUse guard, PreCompact, SessionEnd) now guards on the plugin root, attempts a rate-limited silent bootstrap salvage when the binary is absent (stamp file .bootstrap.attempt, ten-minute window, gitignored), resolves the binary by the command surface's ladder (plugin root first, then PATH), and on continued absence emits one plain actionable line naming what is degraded and the install remedy, with a non-blocking exit — never a raw exec error. Pinned by hooks_selfprovision_test.go (provision, degrade, rate-limit, steady-state, no-root, guard exit fence, PATH fallback); README documents the ladder and the degraded mode. SessionStart keeps its loud primary-provisioner role unchanged."
impact: fix
---

The hook surface has a single point of provisioning and an unactionable failure mode: only the SessionStart chain runs bootstrap.sh, so when that one hook does not fire (or runs without CLAUDE_PLUGIN_ROOT — its guard then exits 0 silently), no other hook ever attempts provisioning and the session is permanently degraded; meanwhile every UserPromptSubmit/PreToolUse/SessionEnd invocation fails as a raw '/bin/sh: .../abcd: No such file or directory' — noisy on every tool call, actionable on none. Observed live in the iss-253 gate failure: hours of hook errors, zero provisioning attempts outside session start, zero SessionEnd transcript captures. Fix direction: each binary-invoking hook guards on the plugin root, attempts a rate-limited silent bootstrap salvage when the binary is absent (bootstrap already has the concurrency lock, bounded curl timeouts, and a one-stat fast path), and on continued absence emits one plain actionable line instead of the exec error. Detector: a hooks-shape test asserting the salvage-and-fallback pattern on every binary-invoking hook, plus an executed-shell fixture test; acceptance: with an empty plugin root and a working bootstrap fixture, the prompt-router hook provisions and runs; with a failing bootstrap it prints one actionable line and exits non-blocking.