---
schema_version: 1
id: "iss-2610160608442201"
slug: "commands-update-md-136-138-gives-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/update.md"
remedy: "Add --deny-self-hosted-runners to the gh attestation verify command at commands/update.md:138."
---

commands/update.md:136-138 gives the by-hand check for an unverified-provenance refusal as gh attestation verify --cert-identity <release.yml on main> --source-ref refs/heads/main --source-digest <the tag's commit>, without --deny-self-hosted-runners. abcd's own policy applies that flag (internal/core/update/update.go:645, hooks/bootstrap.sh), and README.md and docs/how-to/install.md include it, so a person following update.md runs a weaker check than abcd does.
