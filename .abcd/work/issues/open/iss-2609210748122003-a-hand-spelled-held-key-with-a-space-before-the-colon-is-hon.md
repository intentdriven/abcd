---
schema_version: 1
id: "iss-2609210748122003"
slug: "a-hand-spelled-held-key-with-a-space-before-the-colon-is-hon"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "pilot run 2026-09-20, lane C fix round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/provenance.go"
---

A hand-spelled held key with a space before the colon is honoured by every reader and reported by no lint rule. The frontmatter scanner's key regex accepts a space before the colon and the intent loader, intent plan and the record dispatcher all read such a line as a legal hold, but the record_provenance rule sees the key only after the scanner has normalised it, so a line no verb writes (the verb writes held: with no space) passes lint silently; intent unhold refuses it (the remover matches the exact spelling) and sends the caller to repair the line by hand, which is the right refusal but the only signal. Found while applying the review ruling on iss-2609200830076665's fix round. Wanted: record_provenance reports the spelling as a shape no write path produces, the way it reports the disclosure pair, or the scanner exposes the raw key spelling so the rule can see it.
