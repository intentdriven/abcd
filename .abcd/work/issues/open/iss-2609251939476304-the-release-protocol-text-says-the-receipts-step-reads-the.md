---
schema_version: 1
id: "iss-2609251939476304"
slug: "the-release-protocol-text-says-the-receipts-step-reads-the"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release/protocol.go"
---

The release protocol text says the receipts step reads 'the commit the release publishes from', but the release publishes from the tagged merge (runbook step 5 has it right); the content commit is the one the receipts name.
