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

record-lint's prose_citation_resolves refuses a record id whose record exists, minted, on another open pull request's branch and not yet on main: on 2026-11-03 a decision citing dcn-2611031615094092 (in #1381, queued) failed the handover PR's preflight, and the easy way past was to cite the ruling in words, which drops the id and with it the trace (the product thinker ruled ids must always be used). The forward-looking marker keeps the id, but it was written for a record not minted yet, and nothing removes it once the record reaches main, so stale markers accumulate and the next reader cannot tell a pending citation from a dead one. <!-- record-lint: forward-looking -->

The product thinker added (2026-11-03) that a record must be able to reference another that is not yet committed or merged: a decision written in one change often rests on a ruling recorded in another change still in review, and the reference has to carry the id from the start rather than be filled in later by hand. So the gap is also a way to cite a pending record by id that the gates accept now and check again once that record reaches main: resolved then, or reported as a reference to a record that never landed.

And where it is enforced (the product thinker, 2026-11-03): a pending reference holds nothing up when it is written, nor while the record it names is still in review; the release cut is the gate. launch refuses a cut while any reference in the tree names a record that has not reached main (fix it: land that record, or correct the reference), and a pending marker whose record has reached main is reported for removal at the same point, so no release ships a citation to a record that never landed.
