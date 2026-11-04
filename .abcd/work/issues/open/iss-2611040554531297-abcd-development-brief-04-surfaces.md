---
schema_version: 1
id: "iss-2611040554531297"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Name the five events, or refer to the internals section."
---

.abcd/development/brief/04-surfaces/21-update.md:189-192 says the salvage that discards its output is run by the per-prompt, per-command and pre-compaction hooks. hooks/post-tool-use.sh, matched to the question tool, and hooks/permission-request.sh run the same `bootstrap.sh --unseen` (hooks/hooks.json). .abcd/development/brief/05-internals/03-configuration.md:1833-1836 counts five events.
