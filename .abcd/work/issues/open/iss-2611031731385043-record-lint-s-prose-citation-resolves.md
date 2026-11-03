---
schema_version: 1
id: "iss-2611031731385043"
slug: "record-lint-s-prose-citation-resolves"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/prosecitations.go"
remedy: "Resolve a cited id also against the records of open pull requests' branches on origin (named as pending, not missing), and have record-lint flag a forward-looking marker whose id now resolves on main so it is removed; the message teaches citing by id, never by words."
---

record-lint's prose_citation_resolves refuses a record id whose record exists, minted, on another open pull request's branch and not yet on main: on 2026-11-03 a decision citing dcn-2611031615094092 (in #1381, queued) failed the handover PR's preflight, and the easy way past was to cite the ruling in words, which drops the id and with it the trace (the product thinker ruled ids must always be used). The forward-looking marker keeps the id, but it was written for a record not minted yet, and nothing removes it once the record reaches main, so stale markers accumulate and the next reader cannot tell a pending citation from a dead one.
