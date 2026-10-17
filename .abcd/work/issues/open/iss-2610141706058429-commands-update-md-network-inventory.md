---
schema_version: 1
id: "iss-2610141706058429"
slug: "commands-update-md-network-inventory"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.1 release review, docs-currency-reviewer (Opus), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/update.md"
remedy: "Name GitHub attestations and git APIs and the Sigstore TUF repository, reached by update and ahoy install."
---

commands/update.md network inventory (adr-38) says GitHub attestations API and the Sigstore TUF repository are reached by update, to verify build provenance, reading as exhaustive; ahoy install now makes the same check (internal/core/ahoy/apply.go), and both verbs also reach the GitHub git API to resolve the tag commit (internal/core/update/provenance.go), which the inventory omits.
