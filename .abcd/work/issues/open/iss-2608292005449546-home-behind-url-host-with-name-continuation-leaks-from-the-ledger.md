---
schema_version: 1
id: "iss-2608292005449546"
slug: "home-behind-url-host-with-name-continuation-leaks-from-the-ledger"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/adapter/scanner/residual.go"
---

ultra-v0.6.8 follow-up (confirmation pass), corrected by the v0.6.9 combined-diff review: a home behind a URL host or under a longer root with a '.', '-' or '_' continuation — https://ci.example.com/Users/<user>.zip, /Users/<user>-old/x, /Volumes/T7/Users/<user>_snapshot/x — is committed VERBATIM by capture and intent, which have no store-side backstop, and passes through memory and history too: home_path_self declines on nameContinues (which treats '.' and '-' before a letter as a longer name), home_path_other declines at its leading boundary, local_username is URL-suppressed or fails its word boundary, and sweepUserSegment applies the same nameContinues, so the rewriting backstop does not cover these shapes either. This is a REGRESSION against main, which redacted all of them (verified by overlay), introduced by the trailing anchor in 27081ac1 and its follow-ups. Three predicates disagree on '_': nameContinues (boundary), isHomeSegmentByte (continues), isWordRune/wordBounded (continues), and the '_' shape falls through that seam. The branch's tests pin only the '/'-trailing shape behind a host. Fix: one canonical name-continuation predicate for the home-path anchor — alphanumeric only, so '.', '-' and '_' are boundaries; the alnum case (/Users/alexandra vs /Users/alex) is the only false positive the trailing anchor exists for.
