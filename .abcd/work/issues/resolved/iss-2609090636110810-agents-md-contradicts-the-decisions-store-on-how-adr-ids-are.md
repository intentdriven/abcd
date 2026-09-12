---
schema_version: 1
id: "iss-2609090636110810"
slug: "agents-md-contradicts-the-decisions-store-on-how-adr-ids-are"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "sub-agent transcript capture spec authoring"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
resolution: "AGENTS.md's record-ids bullet now states the mint the decide verb performs: abcd decide allocates adr-<yymmddHHMMSS><rrrr> through the shared record-id seam and files it under that stamp, the 0001-0058 ordinals keep their ids and filenames, and no record family needs a word before minting. A test in internal/core/decide mints in a fixture holding the ordinals and then refuses a router that contradicts the id it just minted, so the prose is pinned to the code rather than to another page of prose. The two surfaces that narrate the disagreement historically (the closed spec spc-2609090624222051 and the shipped intent that reported it) are left as written: they record what was true when written."
impact: internal
---

AGENTS.md contradicts the decisions store on how ADR ids are minted, and the stale surface is the one agents read first. The root conventions file states that ADRs keep a hand-numbered filename ordinal and are the one record family where minting from two checkouts still needs a word first. The decide verb's own help says the opposite: the id is a timestamp-numeric stamp allocated through the shared record-id seam, so two branches deciding on the same day cannot collide, with the hand-numbered records grandfathered and every reader admitting both. The contradiction is not inert. In this session it caused a decision to be deferred that was safe to mint, on a coordination risk that adr-45 had already removed, and the deferral was written into a handover note as fact before it was checked. An agent reads the conventions file at the top of every session and reaches the verb's help only if it doubts what it just read.

## Grounds

- pursued: the router is the surface read first, so pinning it to the mint is what stops the deferral recurring; it would be shown wrong if the claim returned in different words, which containment cannot catch and review owns
