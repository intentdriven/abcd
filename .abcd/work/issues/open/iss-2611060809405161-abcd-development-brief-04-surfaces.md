---
schema_version: 1
id: "iss-2611060809405161"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Name `--fill` in the `abcd rules --prompt` paragraph, with what it sets and what happens without it."
duplicates: [iss-2611060809025632]
---

`.abcd/development/brief/04-surfaces/README.md:301` documents `abcd rules --prompt`, in the only brief section the verb has, and does not name `--fill <percent-used>`, which sets the context bracket that form shows. `abcd rules --help` registers the flag, and `docs/reference/cli/commands.md:5645` is its only documented home.
