---
schema_version: 1
id: "iss-223"
slug: "public-visibility-fence-vs-committed"
severity: "minor"
category: "inconsistency"
source: "manual-test"
found_during: "manual-capture"
found_at: "internal/core/ahoy/gitignore.go"
related_issues: ["iss-169"]
details: "Second reproduction 2026-08-15 (later session): a fresh /abcd:ahoy install run reported the absent fence as required .gitignore drift and re-applied it on this repo; while present it gitignored /.abcd/ and silently hid seven uncommitted record files (two intent drafts, five ledger entries) from git status. Reverted again by hand. The fence actively fights the committed-record repo until the visibility table gains a committed-record mode."
related_intents: [itd-159]
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): In the itd-159 planning interview: is the committed-record declaration a switch value or an exception to the public-visibility fence?"
remedy: "Waits on the itd-159 planning interview: if a switch value: the visibility table gains a committed-record value that suppresses the /.abcd/ and /memory/ fence; if an exception: a declared marker in the repo's .abcd/ exempts the fence under the public value; either way a test in internal/core/ahoy proves ahoy install on a committed-record repo writes no fence and reports no drift."
---

the managed .gitignore visibility table has no mode for a public repo that commits its record: on abcd-cli (public, single-repo-curated-release, .abcd/** deliberately in-tree) ahoy install applied the public policy and fenced /.abcd/ and /memory/, contradicting the repo's own boundary that the record is present in every checkout. The visibility table needs a committed-record declaration (config or marker) that suppresses the fence

## Remedy grounds (2026-09-29)

Both forms stop the fence hiding uncommitted record files, which the 2026-08-15 reproduction showed. Rejected: dropping the fence for every public repo, which would publish the record of every public repo that does not mean to commit it.
