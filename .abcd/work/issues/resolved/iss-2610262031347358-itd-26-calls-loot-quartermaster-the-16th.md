---
schema_version: 1
id: "iss-2610262031347358"
slug: "itd-26-calls-loot-quartermaster-the-16th"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 146 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:65"
remedy: "Edit itd-26: drop the '16th agent' ordinal and the AC note that the brief's count of 15 becomes 16; say loot-quartermaster is an addition to the catalog's eighteen (05-internals/01-agents.md), and drop the 'loot-equivalent of lifeboat-oracle' comparison, since agents/ ships no lifeboat-oracle."
related_intents: [itd-26]
finding_ref: cfk-be5c629e212fab75
resolution: "itd-26 no longer calls loot-quartermaster the 16th agent or the loot-equivalent of lifeboat-oracle, and drops the AC note on the count of 15 becoming 16; it now says the agent is an addition to the eighteen the catalog in 05-internals/01-agents.md lists."
impact: internal
resolved_by:
  commit: "09960aa2c81a9feccea11f25c608b9a7826db00c"
---

itd-26 calls loot-quartermaster the 16th agent, while the scope chapter already counts eighteen

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:65` — "(16th agent — provisional name; the loot-equivalent of `lifeboat-oracle`)"
- End B: `.abcd/development/brief/01-product/04-scope.md:32` — "eighteen agents — the canonical roster is the catalog in"

itd-26 introduces loot-quartermaster as the 16th agent. The scope chapter already counts eighteen agents in the canonical roster, so a new agent cannot be the sixteenth. The draft's roster position is stale.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 146.

## Grounds

- pursued: the draft agrees with the scope chapter and the agent catalog on eighteen agents and names no agent agents/ does not ship; TestLootDraftCountsQuartermasterAgainstTheCatalog failing, or the catalog count moving off eighteen, would show it wrong
