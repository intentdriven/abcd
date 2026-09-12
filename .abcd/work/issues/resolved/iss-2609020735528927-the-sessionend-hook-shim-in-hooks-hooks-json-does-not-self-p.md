---
schema_version: 1
id: "iss-2609020735528927"
slug: "the-sessionend-hook-shim-in-hooks-hooks-json-does-not-self-p"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "release-v0.7.1-crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
resolution: "Inverted: the shim is right and the brief was wrong. SessionEnd deliberately never bootstraps (iss-2608210934566223, pinned by TestSessionEndNeverBootstraps); 01-ahoy.md and 05-internals/03-configuration.md carried the same false universal the README shed in iss-2608211432384091, and both now name SessionEnd as the exception. TestTheBriefNamesSessionEndAsTheBootstrapException derives the salvage set from the shipped manifest and holds the chapters to it."
impact: internal
---

The SessionEnd hook shim in hooks/hooks.json does not self-provision: UserPromptSubmit, PreToolUse and PreCompact each carry the bootstrap.sh attempt with the .bootstrap.attempt throttle when the plugin-root binary is missing, and SessionEnd tests the plugin-root binary, falls to the vouched PATH abcd, then fails loudly. On a plugin root with no binary, four events attempt provisioning and SessionEnd silently loses the session transcript, which is the one event whose loss cannot be recovered on the next session. Both 01-ahoy.md (line 225) and 05-internals/03-configuration.md (line 374) document the four non-SessionStart shims as self-provisioning, so the brief describes the intended shape and the shim is the defect. Found by the v0.7.1 brief-surface crosscheck (Direction B, hook entrypoints).

## Grounds

- declined: adding a bootstrap rung to SessionEnd would reintroduce the cancelled-download transcript loss of 2026-08-21; if a later field report shows a binary-less plugin root losing transcripts at exit for want of provisioning, the exception is wrong and a non-blocking salvage would be needed instead
