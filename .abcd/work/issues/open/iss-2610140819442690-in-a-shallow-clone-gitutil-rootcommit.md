---
schema_version: 1
id: "iss-2610140819442690"
slug: "in-a-shallow-clone-gitutil-rootcommit"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the iss-2610090740139804 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Make the root-commit lookup shallow-aware once, in gitutil: have RootCommit (and RootCommitContext) return \"\" for a shallow checkout, which every caller already treats as 'no identity', or add a RootCommitFull that refuses one; prove it with a --depth 1 clone test per caller family, then drop ideate's own RequireFullHistory check in favour of it."
---

In a shallow clone, gitutil.RootCommit (rev-list -n 1 --max-parents=0 HEAD) returns the graft commit, not the repository's root, and every machine-scoped store keyed on it files under that wrong key with no warning: the history ingest (internal/core/history/ingest.go:86), labs (internal/core/lab/lab.go:172), implement runs and loop state (internal/core/implement/load.go:357, internal/core/implement/loop/loop.go:454 and siblings), the report inbox key (internal/core/report/inbox.go:106), the lifeboat probe (internal/core/lifeboat/probe.go:211) and ahoy's managed-repository index (internal/core/ahoy/store.go:68, managed.go:54). A later full clone of the same repository reads a different folder, so the records look lost. CI checks out full history (fetch-depth: 0), so the gates are not affected. Found by review of the ideate private/public fix (iss-2610090740139804), which now refuses a shallow checkout at its own front door.
