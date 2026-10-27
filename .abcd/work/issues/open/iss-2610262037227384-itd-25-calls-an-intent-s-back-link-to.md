---
schema_version: 1
id: "iss-2610262037227384"
slug: "itd-25-calls-an-intent-s-back-link-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 718 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-ab0c8b419a49cf94
---

itd-25 calls an intent's back-link to its issues resolves_issues; 35-drain reads that same back-link as related_issues.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:44` — "the intent's frontmatter gains `resolves_issues: [iss-N, iss-NM, iss-NO]` reciprocally"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:101` — "An intent it was promoted into, one that names it back in `related_issues`, still owns it"

Both name the intent-side field that lists the issues an intent took on, the counterpart of an issue's related_intents. A dredge promotion writes resolves_issues, but the drain decides that an intent still owns a promoted issue by reading related_issues. Under one of the two names, the drain would not see that ownership.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 718.
