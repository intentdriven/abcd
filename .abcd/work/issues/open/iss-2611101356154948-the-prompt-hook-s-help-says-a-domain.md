---
schema_version: 1
id: "iss-2611101356154948"
slug: "the-prompt-hook-s-help-says-a-domain"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Add the periodic refresh to that help paragraph: an item is not sent again unchanged within a session short of a reset or the refresh prompt, every 15th by default or the interval rules.force_refresh_every_n sets, and pin the wording with a help test."
---

The prompt hook's help says a domain, note or page comes back once a session, and not again in another form, until a reset. It does not mention the periodic refresh, which clears both ledgers every 15th prompt by default (rules.force_refresh_every_n in .abcd/config.json), so a matched item also comes back on that prompt.
