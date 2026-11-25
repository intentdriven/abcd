---
schema_version: 1
id: "iss-2611251657223031"
slug: "away-mode-hard-to-find-and-set"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "starting an unattended run on 2026-11-25"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/mode.md"
remedy: "Make bare `abcd mode` say \"away mode: not set\" with the exact command to set it when no away record holds, list the away mode by that name where a person looks for commands, and give it a short how-to page under docs/."
---

A person about to leave cannot tell how to set the away mode or whether it is set. The command is `abcd mode autonomous`, filed under a verb that otherwise sets a waiting-on badge, and its name never says "away". Bare `abcd mode` prints `managed` while no away record holds and says nothing about the away mode, so an unset away mode looks the same as one that does not exist. `abcd mode` has no line of its own in the host command list (`user-invocable: false`), and no page under `docs/` outside the command reference names it. Met on 2026-11-25: a session was opened with a prompt saying the person was away until a stated time and not to wait for them, two phrases the `LEAVING` domain of this tree recalls. No `LEAVING` rule reached the session. The plugin binary whose page the session was served is 0.16.0, whose `abcd rules` has no such domain; it first shipped in v0.17.1. The session stopped for an unrelated reason, said only that the mode was not set, and did not say how to set it, so the person had to ask.
