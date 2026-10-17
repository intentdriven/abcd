---
schema_version: 1
id: "iss-2610142025108700"
slug: "release-receipts-let-deferrals-share-a"
severity: "major"
category: "process"
source: "user-observation"
found_during: "v0.14.1 release receipts"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
remedy: "Close the disposition vocabulary of release receipts to fixed, invalid and deferred, require a deferred finding to name its own open issue record in a record field, refuse a record shared by two findings, and arm it from release-gate manifest schemaVersion 2 (discipline itd-2610141956319695)."
resolution: "From release-gate manifest schemaVersion 2 a release receipt's finding is fixed, invalid or deferred and nothing else, and a deferred finding names its own open issue record in a record field; a missing, closed, symlinked or shared record is refused, in CI and in abcd launch receipts alike"
impact: additive
resolved_by:
  intent: "itd-2610141956319695"
---

Release review receipts accepted any free-text disposition and let any number of deferred findings name one shared catch-all record: 1,090 of the 2,178 findings in committed receipts name iss-2609091956001547, and dispositions came in about twelve free-text shapes, including 'accepted: knowingly shipped unfixed', which defers without saying so. Nothing checked that a deferred finding had a record of its own.
