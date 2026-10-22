---
schema_version: 1
id: "iss-2610152215581665"
slug: "a-reservation-claimed-ahead-of-the"
severity: "minor"
category: "security"
source: "security-advisory"
found_during: "embargo intake 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Date a lease whose claim is ahead of the reader's clock, beyond a small skew allowance, from when that copy first fetched the reservation, so every reservation lapses within the lease maximum of being seen"
resolution: "A lease whose claim time is ahead of the reader's clock, beyond the skew allowance, is dated from when this copy first saw the reservation, so it lapses within the lease maximum of being seen (#1031). Shipped in v0.15.0."
impact: fix
shipped_in: v0.15.0
resolved_by:
  commit: "dcdbafa7f"
---

A reservation claimed with a date ahead of the reader's clock never lapsed.

abcd reserve records a claim time in each reservation commit. When reading a reservation, abcd bounded its lease at a day from the earlier of the claim time and the reader's current time. A claim time set in the future therefore always resolved to a day from the reader's current time, so the reservation never lapsed. The same held, with a two-hour window, for a reservation commit abcd could not read whose committer date was in the future.

Anyone with push access to the repository could push such a reservation and hold an issue or intent indefinitely. Other participants' reserve and take-over were refused, and the protection ruleset abcd ships blocks deleting reservation branches, so abcd offered no way to clear it.

Advisory: [GHSA-p648-6v55-jprh](https://github.com/intentdriven/abcd/security/advisories/GHSA-p648-6v55-jprh), published 2026-10-17 with v0.15.0 as the patched version. The record was held out of the public tree until then.
