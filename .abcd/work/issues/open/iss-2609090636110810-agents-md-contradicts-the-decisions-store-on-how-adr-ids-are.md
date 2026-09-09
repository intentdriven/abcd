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
---

AGENTS.md contradicts the decisions store on how ADR ids are minted, and the stale surface is the one agents read first. The root conventions file states that ADRs keep a hand-numbered filename ordinal and are the one record family where minting from two checkouts still needs a word first. The decide verb's own help says the opposite: the id is a timestamp-numeric stamp allocated through the shared record-id seam, so two branches deciding on the same day cannot collide, with the hand-numbered records grandfathered and every reader admitting both. The contradiction is not inert. In this session it caused a decision to be deferred that was safe to mint, on a coordination risk that adr-45 had already removed, and the deferral was written into a handover note as fact before it was checked. An agent reads the conventions file at the top of every session and reaches the verb's help only if it doubts what it just read.
