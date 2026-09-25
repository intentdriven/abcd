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
resolution: "The roll step names the content commit as the one the reviewers read and every receipt names, and says the release publishes from the tagged merge."
impact: fix
resolved_by:
  commit: "3ab36f809474b925e7dd1f0825766167aafe8d13"
---

The release protocol text says the receipts step reads 'the commit the release publishes from', but the release publishes from the tagged merge (runbook step 5 has it right); the content commit is the one the receipts name.

## Grounds

- pursued: the emitted protocol's roll step says 'tagged merge' and no longer calls the content commit what the release publishes from (TestReceiptsProtocolSaysTheReleasePublishesFromTheTaggedMerge); the old phrase in the step would show it wrong.
