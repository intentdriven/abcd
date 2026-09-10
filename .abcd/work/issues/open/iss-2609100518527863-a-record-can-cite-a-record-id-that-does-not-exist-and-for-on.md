---
schema_version: 1
id: "iss-2609100518527863"
slug: "a-record-can-cite-a-record-id-that-does-not-exist-and-for-on"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs"
---

A record can cite a record id that does not exist, and for one record family an id cannot be resolved to a file mechanically at all. A session writing a spec during an autonomous run put two invented ids into it and caught them only by reading the document back afterwards; nothing in the record gates checks that an id written inside a record resolves to a record that exists. The gap is wider than a missing check. Two filename conventions coexist in the decisions family: the hand-numbered records carry a zero-padded ordinal in the filename whose value differs from the id in their own frontmatter, and the minted ones carry the timestamp id without the family prefix the id itself begins with. Neither shape lets a reader map a cited id to a file by name, so a resolver has to parse the frontmatter of every record in the family before it can answer, and an agent checking its own work by hand cannot do it reliably at all. Any check that lands has to resolve through frontmatter rather than filenames, which is also the reason the check does not exist yet.
