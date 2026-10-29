---
schema_version: 1
id: "iss-2610290507512835"
slug: "abcd-reserve-over-this-copy-s-own-live"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/remote.go"
remedy: "a reserve or take-over that replaces this person's own tip names each path the tip held that the new footprint leaves out (dropped_footprint, a warning on stderr in the text form), saying to name it again to keep holding it"
resolution: "a reserve renewing or taking over this person's own live tip names each held path its footprint neither names again nor covers with a directory (dropped_footprint, a stderr warning)"
impact: additive
---

abcd reserve over this copy's own live reservation writes the new --footprint in place of the held one, so a path the held footprint named and the new one leaves out stops holding off peers, and nothing says it was dropped.
