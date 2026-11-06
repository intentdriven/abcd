---
schema_version: 1
id: "iss-2611060810012471"
slug: "17-guard-md-1038-lists-a-pid-list-a-kill"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Narrow the clause to a pid file written earlier and a variable set before the line, and name `kill-by-search` for the rest."
---

`17-guard.md:1038` lists a pid list a kill reads through a variable or a file, or from a `ps | grep` chain, as unseen. `iss-2611040555239973` and `iss-2611040555220611` record the forms that block as `kill-by-search`. They do not record what stays true: `kill $(cat pidfile)` and a bare `kill $p` still allow, so removing the clause outright would leave the chapter implying those are seen. `p=$(ps aux | grep foo); kill $p` also blocks.
