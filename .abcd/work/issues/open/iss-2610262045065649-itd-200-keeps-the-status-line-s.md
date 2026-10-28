---
schema_version: 1
id: "iss-2610262045065649"
slug: "itd-200-keeps-the-status-line-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1252 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-200, itd-2610030720038073]
finding_ref: cfk-799e2c0526baeed9
---

itd-200 keeps the status line's user-level defaults under ~/.abcd/; itd-2610030720038073 makes an existing ~/.abcd stop every abcd command

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override, and"
- End B: `.abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md:99` — "An existing `~/.abcd` stops every abcd command and hook before it writes anything"

itd-200 places the status line's user-level defaults and off switch under ~/.abcd/. itd-2610030720038073 renames abcd's home to ~/.abcd.noindex and has an existing ~/.abcd stop every abcd command and hook before it writes anything. The location itd-200 states is one abcd now refuses to run beside.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1252.
