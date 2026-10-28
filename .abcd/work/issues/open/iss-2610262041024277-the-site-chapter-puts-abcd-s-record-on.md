---
schema_version: 1
id: "iss-2610262041024277"
slug: "the-site-chapter-puts-abcd-s-record-on"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 973 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/10-site.md:14"
remedy: "none (filed automatically)"
related_intents: [itd-2610091010348892]
finding_ref: cfk-27ae5465ff3e596d
---

The site chapter puts abcd's record on abcdev.app as a surface of this repository; the Pages intent moves abcd's record to its own GitHub Pages route, like any project's.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/10-site.md:14` — "abcdev.app is a surface of this repository and of nothing else."
- End B: `.abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20` — "abcd's own record moves to the same route, built exactly the way any other project's is."

10-site.md describes abcdev.app as the surface of this repository alone, with the record explorer at /record/ and a design target to move the root onto the rendered site. itd-2610091010348892 publishes each opted-in project's record on that project's own GitHub Pages and moves abcd's own record to the same route, leaving abcdev.app as a possible community site. The two give different homes for abcd's record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 973.
