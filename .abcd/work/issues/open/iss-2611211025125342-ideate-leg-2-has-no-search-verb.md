---
schema_version: 1
id: "iss-2611211025125342"
slug: "ideate-leg-2-has-no-search-verb"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "a /abcd:ideate run in the abcd source checkout, 2026-11-20"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Add a read-only search the binary runs over every place leg 2 names, the private verdict store included: it takes the terms and prints, for each, a searches entry ready for the verdict (terms, places, count) and, beside the entry and not inside it, what matched; it says what the count counts (records where a place holds records with ids, files where it holds none); and it exits non-zero when a place cannot be read; the page tells the host to run it and to paste the entries it printed. Grounds: internal, the reason the page itself gives for requiring searches."
---

Leg 2 of /abcd:ideate is searched by hand, and a search that failed reads the same as a search that found nothing. The page names the places to read (the brief, the intents in every bucket, the ADRs, the principles, the research notes, the public verdicts, the decisions and the private verdicts of this machine) and asks for every search as its terms, its places and a count. abcd has no verb that searches them, so the host writes a grep loop of its own and counts by hand. On 2026-11-20 the first loop of a run returned nothing for every term, because the shell passed the list of places as one word; only a zero across all terms being implausible gave it away. `ideate record` refuses a grill with neither hits nor searches, so that an empty grill and a broken search do not write the same record. That refusal is the fix of iss-2610141647307807, which recorded this same failure on 2026-10-14 (a search loop the shell did not split, zero for every term) and was resolved by itd-2610141646454347. This record is what that fix left: a broken search that still reports its terms, its places and a count of zero writes exactly the record an honest empty search writes. What the count counts is not stated either: the page asks how many entries came back, which could be files, lines or records, so the counts of two runs do not compare. A second session reported the same failure from its own run on 2026-11-20: a helper whose list of places the shell did not split, and zero for every place. The recorder cannot tell an entry such a verb printed from one typed by hand, so a host that still writes its own loop, as one served an older page would, records as today; the verb removes the reason to write one and does not close that. This sits beside iss-2608230748418054, the grill's blind spot for records that carry no id.
