---
schema_version: 1
id: "iss-2610230851438977"
slug: "38-announce-md-says-a-hosted-session"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/38-announce.md"
remedy: "Reword: a hosted session abcd's setup prepared runs the committed hooks; only one that skipped the setup runs none, and there the ruleset and the reader are the guards."
refines: [iss-2610171249310666]
---

38-announce.md says a hosted session runs no hook, but the cloud setup script sets core.hooksPath=.githooks (scripts/cloud/connectors/claude/setup.sh:315), so a hosted session abcd set up runs the pre-push hook.
