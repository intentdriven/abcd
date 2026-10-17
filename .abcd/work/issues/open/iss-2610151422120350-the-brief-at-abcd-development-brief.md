---
schema_version: 1
id: "iss-2610151422120350"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:615 to match the shipped surface: The list omits ~/.abcd.noindex/memory.json, which abcd acts on (memory notes recalled into every prompt) and reads through rules.ReadUserLayerFile, the rules layers' trust checks (internal/core/memnote/layers.go)."
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:615 says: Every file in the user scope whose contents abcd acts on — rules.json, trusted-roots, local-transcript-roots, path-entry, cache-attestation, release-mark, config.json, oracle-routing.json, statusline.json, load-limits, credentials.json and credential-homes.json — is refused behind a symlinked ~/.abcd.noindex. The shipped surface differs: The list omits ~/.abcd.noindex/memory.json, which abcd acts on (memory notes recalled into every prompt) and reads through rules.ReadUserLayerFile, the rules layers' trust checks (internal/core/memnote/layers.go). (v0.14.2 release crosscheck, undocumented-surface, raised by opus in a34.)
