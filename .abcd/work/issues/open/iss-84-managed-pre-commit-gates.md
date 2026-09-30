---
schema_version: 1
id: "iss-84"
slug: "managed-pre-commit-gates"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "2026-07-10 prepare-this-repo skill grilling"
related_intents: [itd-62]
deferred_after: "v0.11.1"
deferral_reason: "The product thinker's ruling J23 of 2026-09-29: plan core-owned managed pre-commit gates with draft itd-62. Linked to draft itd-62; ahoy install already scaffolds the banned-names guards (internal/core/ahoy/banlist_scaffold.go), so the secrets and absolute-path gate is what the planning carries. Owed: that draft's planning interview (the draft stays in drafts/ until a person plans it)."
---

Managed pre-commit gates: the interim prepare-this-repo skill offers the secrets and absolute-path pre-commit config from a private template directory outside the repo. The gate belongs to the configuration layer itself: a core-owned surface should install and maintain commit gates in managed repos, replacing the template copy. Seeded from the scaffold-repo retirement analysis; relates to the pluggable safety gate draft (itd-62).