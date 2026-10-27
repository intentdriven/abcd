---
schema_version: 1
id: "iss-2610262039378497"
slug: "itd-105-keeps-every-hook-command-calling"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 874 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-105-a-fresh-plugin-install-just-works-the.md:80"
remedy: "none (filed automatically)"
related_intents: [itd-105]
finding_ref: cfk-600dae779a007a8d
---

itd-105 keeps every hook command calling `$CLAUDE_PLUGIN_ROOT/abcd` directly, with the bootstrap as the first SessionStart hook; the configuration chapter says every event's command is `sh` on a per-event shim and SessionStart is one chained command

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-105-a-fresh-plugin-install-just-works-the.md:80` — "Hook commands keep calling `$CLAUDE_PLUGIN_ROOT/abcd` unchanged."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1291` — "`hooks.json` gives every event the command `sh "${CLAUDE_PLUGIN_ROOT}/hooks/<event>.sh"`, and the shim's logic lives"

itd-105 commits that hook commands keep calling `$CLAUDE_PLUGIN_ROOT/abcd` unchanged, with the bootstrap wired as the first of the SessionStart hooks. The configuration chapter says `hooks.json` gives every event the command `sh "${CLAUDE_PLUGIN_ROOT}/hooks/<event>.sh"`, that the binary is resolved inside that shim, and that SessionStart runs one chained command rather than siblings. A builder following the intent would wire `hooks.json` in a shape the brief has replaced.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 874.
