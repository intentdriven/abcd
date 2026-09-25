---
schema_version: 1
id: "iss-223"
slug: "public-visibility-fence-vs-committed-record-repos"
severity: "minor"
category: "inconsistency"
source: "manual-test"
found_during: "manual-capture"
found_at: "internal/core/ahoy/gitignore.go"
related_issues: ["iss-169"]
details: "Second reproduction 2026-08-15 (later session): a fresh /abcd:ahoy install run reported the absent fence as required .gitignore drift and re-applied it on this repo; while present it gitignored /.abcd/ and silently hid seven uncommitted record files (two intent drafts, five ledger entries) from git status. Reverted again by hand. The fence actively fights the committed-record repo until the visibility table gains a committed-record mode."
related_intents: [itd-159]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): In the merged itd-159 planning interview: is the committed-record declaration a switch value or an exception to the public-visibility fence?"
---

the managed .gitignore visibility table has no mode for a public repo that commits its record: on abcd-cli (public, single-repo-curated-release, .abcd/** deliberately in-tree) ahoy install applied the public policy and fenced /.abcd/ and /memory/, contradicting the repo's own boundary that the record is present in every checkout. The visibility table needs a committed-record declaration (config or marker) that suppresses the fence