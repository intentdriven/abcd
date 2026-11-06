---
schema_version: 1
id: "iss-2611060809025632"
slug: "the-rules-verb-section-at-abcd"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Describe `--fill` under the `--prompt` bullet of the rules verb section."
refines: [iss-2610150526192138]
---

The rules verb section at .abcd/development/brief/04-surfaces/README.md:284-310 lists the bare, `<DOMAIN>` and `--prompt` forms and the diagnostics. `abcd rules` also registers `--fill`, which sets the context bracket `--prompt` shows and is refused at exit 2 without `--prompt` or outside 0 to 100. No chapter under .abcd/development/brief names it; commands/ahoy.md:1745 and docs/reference/cli/commands.md do.
