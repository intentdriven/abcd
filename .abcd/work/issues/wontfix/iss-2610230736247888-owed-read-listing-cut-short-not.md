---
schema_version: 1
id: "iss-2610230736247888"
slug: "owed-read-listing-cut-short-not"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "When the context has expired, mark the reading incomplete through ranOut at the branch listing and at the pull-request listing, and prove it with a test whose context is spent before the read."
wontfix_reason: "Duplicate: the same owed-work reading cut short by its budget was filed as iss-2610241911001655 and fixed on the default branch by 3ef8e5b51, 70335d440 and e52fa47e7, which mark the reading incomplete at the branch listing, the pull-request listing and any git read failing past the deadline."
duplicates: [iss-2610241911001655]
---

The away-mode owed-work reader treats a local-branch listing cut short by its time budget as a plain listing failure: when the budget runs out during git for-each-ref, it notes that the local branches could not be listed but never marks the reading incomplete, so the stop judgement reads it as no own branches and nothing owed instead of failing open on an incomplete reading. A pull-request listing cut short the same way is noted as judged from disk alone. Seen as a failure of TestReadOwedHonoursDeadline in a make preflight run under load on 2026-10-23, and reproduced every time with a context spent before the read.
