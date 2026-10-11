---
schema_version: 1
id: "iss-225"
slug: "ahoy-skill-doc-says-the-path-entry"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "ahoy install dogfood"
found_at: "internal/core"
resolution: "Not an engine defect: the repo-root plugin binary was a month stale (built before iss-171 merged). The current source targets ~/.local/bin via binTarget(); rebuilt via make build and the gap text now matches the doc."
impact: internal
---

ahoy skill doc says the PATH entry defaults to ~/.local/bin (created when absent), but the install engine targeted /usr/local/bin/abcd and stopped when it was unwritable instead of falling back; either the doc or the engine's bin-dir resolution has drifted