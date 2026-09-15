---
schema_version: 1
id: "iss-2608291814576261"
slug: "guard-fail-safe-policy-decided-in-the-cli"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/surface/cli/guard.go"
---

ultra-v0.6.8 altitude 6: the fail-safe policy for a broken repo guard layer (keep bundled hazards armed, drop overrides, exit 1 on allow) is decided in internal/surface/cli/guard.go by inspecting len(reg.Entries) after a guard.Load error, not in core; the plugin hook and any later MCP surface re-derive their own answer and can disagree (ahoy.GuardHealth already carries RepoOverridesDropped separately). guard.Load does return Defaults() with the error, so the dead-branch reading in the review is a stale-base artefact; the placement point stands. Deeper fix: a typed load result in internal/core/guard with the check verb's Decision carrying the drop notice as a warn-level outcome, so surfaces only format it.
