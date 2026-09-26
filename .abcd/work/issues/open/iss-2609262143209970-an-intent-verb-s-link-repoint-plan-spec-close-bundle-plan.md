---
schema_version: 1
id: "iss-2609262143209970"
slug: "an-intent-verb-s-link-repoint-plan-spec-close-bundle-plan"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: drainL sibling sweep of iss-2609261254247117"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
---

An intent verb's link repoint (plan, spec close, bundle plan and close, reclassify, through repointUnderLock in internal/core/intent/lifecycle.go) rewrites every markdown record linking the moved path, issue and reading records included, under the intent store's lock alone: a ledger writer (a resolve, a link, a disposition) landing on a linking ledger record between the repoint's read and its write is erased, and the same holds for a spec record against a spec writer. The intent package cannot take the ledger lock (capture imports it), so closing this needs a registered ledger-lock seam taken OUTSIDE the intent lock (ledger, then intent, the order capture's transition and migrate take), and a stance on a tree with no ledger, which taking the lock would create.
