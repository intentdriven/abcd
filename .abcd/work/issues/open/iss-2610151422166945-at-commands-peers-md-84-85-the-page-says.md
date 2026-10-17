---
schema_version: 1
id: "iss-2610151422166945"
slug: "at-commands-peers-md-84-85-the-page-says"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/peers.md"
remedy: "Correct commands/peers.md:84-85 so it states what the shipped binary does, as the finding describes."
---

At commands/peers.md:84-85: The page says `fetched_at` and `age_seconds` are both absent when nothing has been fetched. The binary emits `"view":{"origin":false,"age_seconds":null}`: reservationView.AgeSeconds has no omitempty, so the key is present as null. A consumer that tests for the key being present gets a different answer from one that tests for null. (v0.14.2 release docs-currency review, raised by opus.)
