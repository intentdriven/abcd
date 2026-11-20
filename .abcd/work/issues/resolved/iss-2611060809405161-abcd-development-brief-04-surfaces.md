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
resolution: "The rules verb section's --prompt bullet names --fill <percent-used>, says it sets the context bracket shown, and says that without it the fill is unknown and the unknown-fill form renders (prose landed in 920004d5 for the duplicate iss-2611060809025632); TestTheRulesBriefsPromptBulletDescribesFill now pins both facts."
impact: internal
resolved_by:
  commit: "267a8fe40574bf6dc6ad2094bfa3496a64c01ea4"
---

`.abcd/development/brief/04-surfaces/README.md:301` documents `abcd rules --prompt`, in the only brief section the verb has, and does not name `--fill <percent-used>`, which sets the context bracket that form shows. `abcd rules --help` registers the flag, and `docs/reference/cli/commands.md:5645` is its only documented home.

## Grounds

- pursued: the brief's --prompt bullet states what --fill sets and what renders without it, matching docs/reference/cli/commands.md and the flag's help; a bullet missing either fact fails TestTheRulesBriefsPromptBulletDescribesFill, which was watched fail against the brief as it stood before 920004d5.
