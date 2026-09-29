---
schema_version: 1
id: "iss-2609282105240081"
slug: "abcd-banlist-list-with-no-flags-prints-output-byte-identical"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "v0.11.1 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
deferred_after: "v0.11.1"
deferral_reason: "The naming chapter forbids the shape but prescribes no remedy, and each remedy changes a shipped verb: making an unfiltered banlist list refuse and point at bare banlist (the capture list precedent) breaks a script that calls it, retiring list moves its layer filter onto bare banlist, and recording it as an exception keeps the shape; which one is the product thinker's choice, so the record waits for that ruling rather than a lane picking one (rulings-owed AY)"
---

`abcd banlist list` with no flags prints output byte-identical to bare `abcd banlist`, the alias shape the naming chapter (02-constraints/04-naming.md) forbids, and no exception lists it. Found by the v0.11.1 crosscheck (x-115); sub-verb registered at v0.11.0.

## Deferral 2026-09-28

Deferred past v0.11.1: The naming chapter forbids the shape but prescribes no remedy, and each remedy changes a shipped verb: making an unfiltered banlist list refuse and point at bare banlist (the capture list precedent) breaks a script that calls it, retiring list moves its layer filter onto bare banlist, and recording it as an exception keeps the shape; which one is the product thinker's choice, so the record waits for that ruling rather than a lane picking one.
