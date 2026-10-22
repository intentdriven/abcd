---
schema_version: 1
id: "iss-2610222040450844"
slug: "the-rulesets-readme-abcd-work-rulesets"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "lane E (abcd-54) preparing the PT hosted run, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/README.md"
remedy: "replace the 'Not yet applied' sentence with the date it was applied and its id, add 25302453 to the README's id list, and diff status-protection.json against the live ruleset so the mirror is confirmed rather than assumed"
---

The rulesets README (.abcd/work/rulesets/README.md) still says status-protection.json is 'Not yet applied' and that its id joins the list below once it is, but the ruleset has been live since 2026-10-17 as 'status protection' (id 25302453, enforcement active, read through the API on 2026-10-22), so the committed mirror's documentation contradicts the forge
