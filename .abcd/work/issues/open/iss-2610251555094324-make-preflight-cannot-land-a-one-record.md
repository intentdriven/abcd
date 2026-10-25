---
schema_version: 1
id: "iss-2610251555094324"
slug: "make-preflight-cannot-land-a-one-record"
severity: "major"
category: "process"
source: "user-observation"
found_during: "pushing the capture branch for iss-2610241228359748, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Let preflight mint a receipt without the test and race stages for a diff that touches only the record tiers (.abcd/work/, .abcd/development/), using the same fail-closed path classifier CI uses to stand the race lane down on a pull request confined to those paths, so the receipt's scope matches what CI would run."
refines: [iss-2610161639025702]
---

make preflight cannot land a one-record change on a busy machine: on 2026-10-24 a branch whose whole diff is one new issue record (iss-2610241228359748) ran preflight twice, every gate before the test stages passed both times, and both runs then waited the load gate's full two hours (abcd implement load --wait --timeout 2h) behind two to four running preflights and six to eight queued waiters, and failed with 'load gate still shut after 2h0m0s'. With no receipt the pre-push hook refuses the push, so a change that touches no Go code waits behind the full test and race stages of every other session and in practice cannot be pushed.
