---
schema_version: 1
id: "iss-327"
slug: "the-ship-flow-permits-an-unreleasable-cut-launch-ship-writes"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/surface/cli/ship.go"
related_intents: [itd-93]
resolution: "folded into itd-93's scope"
impact: internal
---

The ship flow permits an unreleasable cut: launch ship writes the dated heading with no check that the receipts protocol can complete, so the receipt gate fires at tag time — the most expensive possible moment, unrecoverable once tagged. Folded into itd-93 by maintainer decision: the emit prints the receipts protocol as a checklist, the ingest refuses a cut it cannot prove releasable, and a launch receipts sub-verb runs the release job's exact check locally before the merge