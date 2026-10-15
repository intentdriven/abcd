---
id: dcn-2609150000000003
slug: banned-names-get-both-halves-product
date: 2026-09-15
---
2026-09-15 — Banned names get both halves (product thinker, ruling on iss-2609100506269348, choosing "both" once told a home list is invisible to CI): a committed declaration lifts the public visibility fence so a fresh public repository can create its committed banned-names layer on its first commit, and a machine-global private list in the user-level home bans a name in every repository on the machine. CI never reads the home list and nothing from it reaches a committed file. Filed as itd-2609151516525843 (`builds_on` itd-74, `refines` adr-56); whether the home is one macOS user's or shared across users stays the product thinker's separate open question. Rejected: move the banlist config out of the hidden directory; make the public list private-visibility only.
