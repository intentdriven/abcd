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
resolution: "The rules verb section's --prompt bullet now describes --fill: the share of the context window used, 0 to 100, which sets the bracket shown; the unknown-fill form without it; and its refusal at exit 2 without --prompt or outside that range."
impact: internal
resolved_by:
  commit: "920004d526ac45a1874098b14000c7624cb4dfc9"
---

The rules verb section at .abcd/development/brief/04-surfaces/README.md:284-310 lists the bare, `<DOMAIN>` and `--prompt` forms and the diagnostics. `abcd rules` also registers `--fill`, which sets the context bracket `--prompt` shows and is refused at exit 2 without `--prompt` or outside 0 to 100. No chapter under .abcd/development/brief names it; commands/ahoy.md:1745 and docs/reference/cli/commands.md do.

## Grounds

- pursued: the brief's --prompt bullet matches the --fill refusals in internal/surface/cli/cli.go and the wording in commands/ahoy.md and docs/reference/cli/commands.md; a CLI that accepted --fill without --prompt or outside 0 to 100, or a bullet missing any of those facts (TestTheRulesBriefsPromptBulletDescribesFill), would show it wrong.
