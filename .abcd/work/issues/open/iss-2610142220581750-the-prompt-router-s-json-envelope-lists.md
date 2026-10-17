---
schema_version: 1
id: "iss-2610142220581750"
slug: "the-prompt-router-s-json-envelope-lists"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "spc-2610111627213138 security round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Add the layer to each entry of the hook's --json recalled list (or a sibling recalled_layers map) in a change that versions the envelope, matching the plain stderr line's \"mem-… (project)\" labels."
---

The prompt router's --json envelope lists recalled notes as bare ids with no layer, while the injected text and the stderr line both name each note's layer (project or machine). A machine reader of the envelope cannot tell a note from the repository's committed .abcd/memory.json from the person's own. It was left as is because adding the layer changes the envelope's schema.
