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
---

`abcd banlist list` with no flags prints output byte-identical to bare `abcd banlist`, the alias shape the naming chapter (02-constraints/04-naming.md) forbids, and no exception lists it. Found by the v0.11.1 crosscheck (x-115); sub-verb registered at v0.11.0.
