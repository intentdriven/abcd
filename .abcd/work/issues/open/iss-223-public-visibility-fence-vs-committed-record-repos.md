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
promoted_to: itd-159
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (promoted to draft itd-159 (not ready): planning owed for a committed-record declaration that suppresses the public-visibility fence). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

the managed .gitignore visibility table has no mode for a public repo that commits its record: on abcd-cli (public, single-repo-curated-release, .abcd/** deliberately in-tree) ahoy install applied the public policy and fenced /.abcd/ and /memory/, contradicting the repo's own boundary that the record is present in every checkout. The visibility table needs a committed-record declaration (config or marker) that suppresses the fence