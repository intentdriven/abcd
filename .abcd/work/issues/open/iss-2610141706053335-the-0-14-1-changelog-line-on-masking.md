---
schema_version: 1
id: "iss-2610141706053335"
slug: "the-0-14-1-changelog-line-on-masking"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.1 release review, docs-currency-reviewer (Opus), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Change capture status to bare capture in the resolution text of iss-2610121653499543; the released 0.14.1 line stays as published."
---

The 0.14.1 changelog line on masking names the reasons, titles and records that drain, capture list and capture status show, but abcd has no capture status sub-verb: bare abcd capture renders the ledger status, and abcd capture status gets the lone-word refusal. The wording comes from the resolution text of iss-2610121653499543, which the changelog derives from.
