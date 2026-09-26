---
schema_version: 1
id: "iss-2609261909106167"
slug: "the-private-banlist-guard-in-githooks-pre-commit-reads-every"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainS3"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
---

The private-banlist guard in .githooks/pre-commit reads every staged blob raw under LC_ALL=C, so a private name written with JSON string escapes passes it: a \uXXXX spelling of any letter, non-ASCII or plain ASCII, or a \/ inside a pattern, puts bytes in the blob that no ERE written for the plain spelling matches, and a JSON transcript, export or fixture is the natural carrier. The store-before-commit redactors and the lint rules read the scanner's decoded views (lineViews: the percent view and the JSON escape layers); the hook, the one enforcement point of the private layer, reads only the text as written. Detector: a keyed banlist entry for a fake name refuses a staged JSON file that spells the name with \u escapes or percent-encoding, exactly as it refuses the plain spelling.
