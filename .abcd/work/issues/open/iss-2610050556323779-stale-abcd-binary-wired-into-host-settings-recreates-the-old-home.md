---
schema_version: 1
id: "iss-2610050556323779"
slug: "stale-abcd-binary-wired-into-host-settings-recreates-the-old-home"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "/abcd board session; the GropiusLLM session (gropiusllm-a3) reported the same symptom"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy"
remedy: "Have abcd ahoy read the host's user settings (read-only) and report every command that invokes an abcd binary other than the one ~/.abcd.noindex/path-entry records, naming the file, the event and the remedy, and fail the status-line check the same way; abcd never edits that file to remove it."
---

On 2026-10-05, after the person had renamed ~/.abcd to ~/.abcd.noindex, ~/.abcd reappeared three times within an hour, and each time the guard refused every shell command and every question in two sessions at once (this one and gropiusllm-a3), because one stray folder blocks every session on the account. The cause was a SubagentStop entry in the person's ~/.claude/settings.json that ran a locally built ~/ABCDevelopment/abcd/bin/abcd-darwin-arm64 (old enough to reject --version) with 'hook subagent-stop'. Whenever any subagent stopped, it staged that subagent's transcript under ~/.abcd/transcripts/<root-sha>/staging/, recreating the old home. Evidence: the stray folder held only transcripts/488a0aa9…/staging/20261005T054848.953201000Z-a765320031b9991b5.stage.json and .raw (a subagent of this session, staged at 05:49:05Z) and an empty sessions/ marker. The plugin already registers its own SubagentStop hook with the current binary, so the settings entry was a stale duplicate. Removing it and moving the staged files into ~/.abcd.noindex stopped the recurrence. The rename's guard (itd-2610030720038073, D2) only notices the old folder after something has written it. Nothing in abcd, ahoy included, notices an abcd command wired into the host's settings outside the plugin, so a binary built before a migration keeps writing the old layout in every session until a person hunts it down by hand.
