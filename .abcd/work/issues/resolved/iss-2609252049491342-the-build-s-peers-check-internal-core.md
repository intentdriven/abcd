---
schema_version: 1
id: "iss-2609252049491342"
slug: "the-build-s-peers-check-internal-core"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "peersCheck counts every peer the listing names and cannot read, whose holding is unknown (peers.Report.Unjudged), as a holding, as it does an unreadable claim; a foreign-repository peer and a pre-layout peer do not count. TestStartRefusesAPeerItCannotRead and TestUnjudgedNamesThePeersWhoseHoldingIsUnknown."
impact: fix
resolved_by:
  commit: "8911d088"
---

The build's peers check (internal/core/implement/loop/check.go peersCheck) fails open on one side and closed on the other: a peer the peer listing names and does not read (Peer.NotRead: its directory or git refuses, its record folders cannot be listed, its ledger holds one id in two folders) is skipped silently by peers.Report.Locate, so a peer whose holding is unknown reads as holding nothing, while the claim half refuses an Unreadable claim as a holding. A record the check cannot see is not one it may start past; the unread peer should refuse as the unreadable claim does, naming it.

## Grounds

- pursued: we expect a build to refuse, naming it, while any peer it cannot read exists; shown wrong if an unread peer of this repository lets a start through, or an old-layout branch blocks one
