---
schema_version: 1
id: "iss-321"
slug: "persona-pronoun-discipline-itd-79-is"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/intents"
resolution: "seven intents rewritten to they/them with verb agreement per itd-79."
impact: internal
---

persona pronoun discipline itd-79 is violated by gendered pronouns in seven intent records including a shipped intent
## Evidence
itd-79 (`intents/disciplines/itd-79-persona-registry.md:18`, also personas.json and intents/README.md:241): "Every persona is referred to as they/them." Violated (gendered pronouns) in: `shipped/itd-100:17,25`; `drafts/itd-126:17,29,31`; `drafts/itd-97:32`; `drafts/itd-92:20`; `drafts/itd-98:28`; `drafts/itd-108:48`; `drafts/itd-128:23` (post-discipline, authored 2026-08-19).

## Adversarial verdict: CONFIRMED (nitpick)
itd-79:32 names "non-attribution mentions" as in-scope, review-enforced (the persona_registry lint matches only `said <Name>,`, so a regex cannot reach pronouns — these slipped the review half, not a linter gap). Grandfathering refuted (itd-128 postdates the rule). docs/brief/principles carry zero gendered pronouns — these seven are the outliers. Fix requires verb-agreement edits, not a blind sed (e.g. "she has to"→"they have to", "he edits"→"they edit"). Exclude the itd-27:74 "Pocock/his" — a real prior-art author citation, not a persona. record-lint stays green (no attribution name changes). itd-79's own severity is minor; this is the review-tier half.
