---
schema_version: 1
id: "iss-2610160608359152"
slug: "the-design-target-callout-at-abcd"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md"
remedy: "Rewrite the callout at 08-abcd.md:478-484 to say only the itd-20 rows (visibility, disembark log, dev-sync record, logbook) were never built, not that the board reads nothing else."
---

The design-target callout at .abcd/development/brief/04-surfaces/08-abcd.md:480-482 says the shipped status path reads .git, .abcd/development and the three work tiers 'and nothing else: ... no linked intents, no spec store, and no thresholds'. The shipped board the same chapter documents (lines 153-342) also reads the mode state, peers, the inbox, oracle routing, the reviews block with its twenty-commit threshold (internal/core/reviews StaleAfter = 20) and the planned intents with their specs (internal/core/statusblock).
