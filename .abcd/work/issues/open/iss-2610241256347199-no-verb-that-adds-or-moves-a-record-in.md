---
schema_version: 1
id: "iss-2610241256347199"
slug: "no-verb-that-adds-or-moves-a-record-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "records branch for itd-2610240922461281, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/indexgen.go"
remedy: "Have each verb that writes or moves a record in a directory an index_drift pair enumerates (the draft filer, capture promote, intent plan, spec close, decide --adr) call lint.RegenerateIndexes for the regions it touched and report the rewrite, with a test per verb that record-lint's index_drift stays clean after it runs; a hand-filed record still needs go generate, which the index_drift finding should name as its fix."
---

No verb that adds or moves a record in an indexed directory regenerates the index that lists it. Only go generate ./internal/surface/cli (cmd/abcd-gen-surface, through lint.RegenerateIndexes) rewrites the generated regions, such as the drafts list in .abcd/development/brief/06-delivery/03-out-of-scope.md and the ADR index. So filing a draft intent, planning one out of drafts/, or minting an ADR with decide --adr leaves a region stale, and record-lint's index_drift blocker refuses the next make preflight. On 2026-10-24 a hand-filed draft (itd-2610240933025076) failed preflight this way. The region then conflicted on a rebase over main, because every branch that files a draft edits the same list.

## Further evidence 2026-10-26

- A records-only pull request conflicted with main on the forge twice in one day, each time only in these generated regions (the drafts list in the out-of-scope page and the ADR index), because another branch that filed a draft or an ADR merged first. Each conflict turned auto-merge off and needed a local merge, and the merge commit needed a fresh `make preflight`, which on this machine waited three to five hours at the load gate. A union merge driver would not help: as the root `.gitattributes` says, the forge computes mergeability and the queue's merge without it.
- So the cost of committing a generated list that every record-filing branch edits is a full preflight per collision, not a small rebase. A remedy that removes it: keep the regions out of the branch's diff. Either generate them on the merge queue's tree and check them there instead of committing them on each branch, or render the lists at read time from the folders and drop the committed regions.
