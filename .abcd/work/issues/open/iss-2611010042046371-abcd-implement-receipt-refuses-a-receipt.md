---
schema_version: 1
id: "iss-2611010042046371"
slug: "abcd-implement-receipt-refuses-a-receipt"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "driving build runs overnight"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/implement.go"
remedy: "Infer the run from the receipt path when it lies under a run directory of this checkout, and refuse only when the path names no run or names one that differs from --run."
---

abcd implement receipt refuses a receipt with 'N runs are in progress ... name one with --run' whenever more than one run is open in the checkout, although the receipt path it is given already sits under .abcd/.work.local/run/<run-id>/ and so names its run; on 2026-11-01 each of three concurrent runs' receipts had to be handed back a second time with --run.
