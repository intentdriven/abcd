---
schema_version: 1
id: "iss-2610241544441841"
slug: "the-default-branch-protection-s-required"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "verifying forge settings, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/main-protection.json"
remedy: "Either add wip-guard to the live main protection ruleset's required checks (the mirror and ci.yml say it is required) or remove it from the mirror and the comment, as the person rules; then refresh the mirror with the rulesets README's command so the two agree."
---

The default-branch protection's required checks differ between the committed mirror and the live ruleset: .abcd/work/rulesets/main-protection.json and the comment in ci.yml require wip-guard, but the live ruleset (id 19969675) requires nine checks without it, matching the rulesets README's count. The live ruleset was last changed on 2026-10-11, before the commit that added wip-guard to the mirror on 2026-10-15, so the mirror records a requirement GitHub does not enforce.
