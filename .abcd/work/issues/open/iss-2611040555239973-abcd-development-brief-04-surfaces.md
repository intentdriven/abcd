---
schema_version: 1
id: "iss-2611040555239973"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Remove those shapes from the unseen list and name kill-by-search as what blocks them."
---

.abcd/development/brief/04-surfaces/17-guard.md:1013-1014 says an allow does not see a pid list a kill reads through a file or from a `ps | grep` chain. `ps aux | grep make | xargs kill`, the awk and `kill $(...)` forms and `pgrep make > p; kill $(cat p)` all exit 1 under kill-by-search. commands/guard.md:992 states that a filtered ps listing is a search.
