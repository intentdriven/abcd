---
schema_version: 1
id: "iss-2610161555138660"
slug: "every-abcd-hook-shows-the-status-message"
severity: "nitpick"
category: "ux"
source: "user-observation"
found_during: "crash recovery on 2026-10-16 (abcd-22/abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "Give each hook a status message that says what it does (for example 'abcd: loading rules and status' for UserPromptSubmit). Mention the download only where the bootstrap actually runs, or drop it from hooks.json, since the shim decides at run time."
---

Every abcd hook shows the status message 'abcd: checking the plugin binary; a first run downloads it, so this can take a while', even when the binary is already installed and the hook only runs the prompt router. When the router timed out on 2026-10-16, that message pointed the person at a download that was not happening.
