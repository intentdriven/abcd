---
schema_version: 1
id: "iss-2611151241442197"
slug: "a-spec-close-ships-an-intent-whose-text"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "an autonomous run's spec close"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/spec/store.go"
remedy: "Have spec close run the site's banned-token check over the intent it ships and refuse naming the line and rule, and have the preflight's site-render step print the site lint's findings instead of discarding them to /dev/null."
---

A spec close ships an intent whose text the site check then refuses, and nothing tells the person until the full preflight's site render fails. Closing the sessions-announce spec moved its intent into the shipped set the site renders; the intent's planning provenance note says it was renamed by a decision, and the site lint's present-tense rule refused that phrase, so the preflight failed at site-render with only 'exit status 1' and the finding had to be found by running the site build and lint by hand. The spec close passes record-lint and docs-lint, which do not read shipped intents through the site's banned-token rules.
