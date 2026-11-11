---
schema_version: 1
id: "iss-2611101321036534"
slug: "a-release-cut-does-not-read-how-much"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/ship.go"
remedy: "Waits on a ruling first: then have launch read the usage left before a cut and set it against the cost of past cuts, and give a cut's handover a place abcd writes and the other account reads."
---

A release cut does not read how much usage the account has left before it starts, and nothing hands a cut to another account except a note written by hand. On 7 Nov the other account's usage ran out before it could cut v0.17.0, with about 5 per cent left until the following Wednesday, after a day of fixes in the release's scope; the handover was a note in a shared folder, and its follow-up list could not be written to from this account. A person must choose whether a cut refuses or warns on low usage, and where a cut's handover to another account lives.
