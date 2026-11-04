---
schema_version: 1
id: "iss-2611040555142660"
slug: "06-delivery-02-verification-matrix-md-98"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Limit the statement to the reservation writes and name the sweep as the one deleter."
---

06-delivery/02-verification-matrix.md:98 says a renewal, take-over and release are each one child commit and nothing is ever deleted or forced. abcd reserve --sweep --apply deletes spent reservation branches, .github/workflows/reservation-sweep.yml runs it on a schedule, and 04-surfaces/37-reserve.md:58-71 documents that as the verb's fourth form. No matrix row mentions the sweep.
