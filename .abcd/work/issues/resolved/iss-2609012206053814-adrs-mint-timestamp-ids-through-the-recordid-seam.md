---
schema_version: 1
id: "iss-2609012206053814"
slug: "adrs-mint-timestamp-ids-through-the-recordid-seam"
severity: "major"
category: "process"
source: "user-observation"
found_during: "design-decisions-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs"
resolution: "`abcd decide \"<title>\"` mints adr-<yymmddHHMMSS><rrrr> through recordid.Minter and writes the ADR skeleton to decisions/adrs/<stamp>-<slug>.md; recordid.CanonADRID and recordid.ADRFileID are the one derivation every reader of an ADR id or filename takes, so the ordinals 0001-0058 and the minted stamps resolve, lint, sort and dispatch side by side, with ordinals ahead of stamps in the index order."
impact: additive
resolved_by:
  commit: "901709264eba31d6e34c6743c49dd2d357304bfa"
---

Ruled 2026-09-01 (DECISIONS.md, that date): ADRs take timestamp ids through the same recordid seam as issues, because two branches minted 0055 and 0056 on the same day for different decisions. Implementation owed: a verb that mints adr-<timestamp> and writes the ADR skeleton under decisions/adrs with a filename ordered by the stamp; record-lint and the wiki-link resolver admit both the ordinal shape (0001-0058, kept as is) and the timestamp shape; the ADR index and site export sort ordinals first then stamps; adr-45's rollout note names this as the deferred family. Companion to the intent/spec adoption tracked by iss-2608210737260468.

## Grounds

- pursued: an ADR id allocated by reading the store collides across branches by construction, so moving the family onto the coordination-free mint removes the collision rather than warning about it; falsified if two checkouts minting in the same second still converge on one id, or if any reader of an ADR id refuses either vintage
