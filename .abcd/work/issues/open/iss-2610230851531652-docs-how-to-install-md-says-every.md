---
schema_version: 1
id: "iss-2610230851531652"
slug: "docs-how-to-install-md-says-every"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/install.md"
remedy: "Name the hooks that do not bootstrap and add the stop check to the list in docs/how-to/install.md."
---

docs/how-to/install.md says every live-session hook needing the binary tries the bootstrap, but stop.sh, stop-failure.sh and subagent-stop.sh never do, and its list of what a broken install loses omits the away mode's stop check.
