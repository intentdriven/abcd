---
schema_version: 1
id: "iss-2610262044079779"
slug: "itd-87-escalates-a-finding-that-matches"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1184 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:17"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): When a consistency finding's stable key matches an already resolved or wontfix record, does the pass mark it addressed and file nothing (G1), or file a new entry citing the closed one as a recurrence (itd-87)? Recommend: for a resolved record, surface it as a recurrence and file the escalation, because identical quoted text still present means the fix did not hold (recurrence-is-signal); for a wontfix record keep addressed, since a person already ruled on that text."
related_intents: [itd-87, itd-2609291923559186]
finding_ref: cfk-489645189d2b62c6
---

itd-87 escalates a finding that matches a resolved or wontfixed record as a new entry; the consistency pass marks such a finding addressed and files nothing

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:17` — "the ledger no longer quietly folds it into the closed entry. It escalates: a new entry, citing the closed one"
- End B: `.abcd/development/intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md:87` — "fix is marked `addressed` on the report, naming the record, and files"

itd-87 says a finding that matches an entry already resolved or wontfixed must not be folded into the closed entry: it becomes a new entry citing the closed one, because the recurrence is evidence that the closure may be wrong. itd-2609291923559186 G1 says that when a consistency finding's record is already resolved or won't-fix, the report marks it `addressed` and files nothing. The two give opposite rules for the same event in the issue ledger.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1184.
