---
schema_version: 1
id: "iss-2610240823574047"
slug: "ahoy-install-run-from-a-plain-terminal"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "ahoy install from a terminal, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/statusline_guard.go"
remedy: "check for a trusted PATH entry before the status-line offer is asked, and when none can be written offer the README one-liner (or resolve the plugin root from the plugin cache) inside the same run instead of refusing after the answers"
---

ahoy install run from a plain Terminal cannot wire the status line or a PATH entry: with no CLAUDE_PLUGIN_ROOT the plugin root is missing (plugin.root_missing), stepPathEntry returns early, ~/.abcd.noindex/path-entry stays absent, and the status-line offer is answered yes through all its element switches only to be refused afterwards (no PATH entry on record). The person learns of the prerequisite after answering every question.
