---
schema_version: 1
id: "iss-2611060809045160"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Add a section to the setup chapter stating what `abcd ahoy shared-folder` does, what it writes and when it refuses."
---

`.abcd/development/brief/04-surfaces/01-ahoy.md:34` lists `abcd ahoy shared-folder` as shipped, and the chapter says nothing else about it beyond the generated appendix (:1587, :1671). What the verb is for, what it writes and what `--person` means are stated only in `commands/ahoy.md:1475` and in the verb's help; `.abcd/development/brief/04-surfaces/17-guard.md:207` and `.abcd/development/brief/04-surfaces/27-implement.md:697` mention the folder in passing.
