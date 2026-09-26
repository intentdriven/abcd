---
schema_version: 1
id: "iss-2609260100391018"
slug: "the-pretooluse-hook-wrapper-in-hooks-hooks-json-says-shell"
severity: "nitpick"
category: "ux"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
---

The PreToolUse hook wrapper in hooks/hooks.json says 'shell commands run UNGUARDED' when the binary is missing or crashes, including on a call to the host's question tool, where it is the question that runs ungated, not a shell command.
