---
schema_version: 1
id: "iss-2610241107191205"
slug: "a-lapsed-hold-or-reservation-must-be"
severity: "major"
category: "process"
source: "user-observation"
found_during: "drain session abcd-0d, 2026-10-24, after a crashed computer's leases lapsed"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Make lapse autonomous end to end: every surface that relays a hold to an agent (the prompt hook's status notes, which show each hold once and never say when it lapses, the landing's notice, and the merge-hold guidance in the CONCURRENCY rules and /abcd:launch) acts only on a live hold and says when it lapses; the CONCURRENCY rule tells an agent to re-read a hold's expiry immediately before acting on it or relaying it; and spent reservation branches leave origin unattended, through PR #1111's scheduled sweep once its App, environment and ruleset bypass are set up. How status and peers name a lapsed release cut is a peer's separate capture."
---

A lapsed hold or reservation must be lifted without a person, or autonomous work waits on a holder that no longer exists. On 2026-10-24 a computer that had reserved a release cut (rel-v0.15.2) and dozens of issues and intents crashed. Its leases lapsed, yet sessions on another machine kept waiting on it: one session relayed the release hold to six others as live, from a read made before it lapsed, two finished pull requests waited for the dead cut's ruling, and a build was held behind the crashed copy's work. The product thinker ruled the same day that lifting a lapsed hold is autonomous: no person is needed to declare a lapsed holder gone. The reading side already counts a reservation five minutes past its expiry as released. What is missing is that every reader, verb and agent instruction acts on that, and that spent branches leave origin without a person. This record carries the ruling that the whole path runs unattended.

How status and peers name an abandoned cut is a peer session's capture, iss-2610241104098581, which was not on origin when this was written. <!-- record-lint: forward-looking -->
