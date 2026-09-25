---
schema_version: 1
id: "iss-139"
slug: "skill-md-compatibility-watch"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-100 terminology crosswalk"
found_at: ".abcd/development/brief/05-internals/08-skills.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Park the findings-only skill until the SKILL.md spec is versioned and stewarded, or ship one now?"
---

SKILL.md compatibility watch: the record rules abcd ships zero skills (commands-only namespace; any state mutation is a command). The external Agent Skills spec (agentskills.io) is unversioned, draft-stage, and without a standards-body steward, but multi-vendor adoption is real. If a findings-only skill ever ships, SKILL.md is the accepted executable form — this capture tracks the spec's maturation (versioning, stewardship, the Experimental fields) so that decision is made against current facts. The itd-100 crosswalk's REJECTS row cites this id.