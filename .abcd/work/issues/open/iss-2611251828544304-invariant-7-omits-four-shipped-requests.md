---
schema_version: 1
id: "iss-2611251828544304"
slug: "invariant-7-omits-four-shipped-requests"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md"
remedy: "Name each in invariant 7 with the act or opt-in that covers it, or remove the request; take the sentence that says none exists out once one is named. Grounds: an invariant a reader trusts must list what code does."
---

Brief invariant 7 says no version-discovery request exists anywhere in abcd and lists what the network is asked for, and four requests that shipped are not in it: the resolve of the newest release by the bootstrap from a hook (itd-132), the background git ls-remote of the dogfood build (opted in per machine), the hourly ask about a newer release by a drive on its yes (itd-2611062149308723), and the detached ask of the forge about scheduled runs that the prompt hook starts (the status chapter). Checked on 2026-11-25 by reading the invariant for each name.
