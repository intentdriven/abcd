---
schema_version: 1
id: "iss-2611101229244628"
slug: "the-statusline-verb-s-one-line-sentence"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "spc-2610111627213138 step 7 docs lane"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
remedy: "Reword the sentence to name the one write (the session's fill file) and regenerate the surface manifest and CLI reference with go generate ./internal/surface/cli"
---

The statusline verb's one-line sentence still says it writes nothing (internal/core/surface/sentences.go, mirrored in .abcd/development/release/surface.json and docs/reference/cli/commands.md), though since spc-2610111627213138 step 5 the verb writes the session's fill file under ~/.abcd.noindex/fill/ on the path that renders abcd's own row, as its own long help says.
