---
schema_version: 1
id: "iss-2610262031305231"
slug: "the-naming-chapter-stages-abcd-audit"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 138 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:117"
remedy: "none (filed automatically)"
related_intents: [itd-16]
finding_ref: cfk-77c024173c377d87
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030497817. Same register line against the same fact (itd-35 owns the lifeboat sub-verb); only the quoted second end differs (itd-16 itself rather than the intent chapter)."
duplicates: [iss-2610262030497817]
---

The naming chapter stages /abcd:audit lifeboat under itd-16, while itd-16 assigns that sub-verb to itd-35

A scope leakage (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:117` — "`/abcd:audit lifeboat` (itd-16)"
- End B: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:23` — "Sibling sub-verbs (`/abcd:audit lifeboat <path>` per itd-35; future audit applications)"

The naming chapter lists /abcd:audit lifeboat as a staged sub-verb belonging to itd-16. itd-16 itself names the sibling sub-verb /abcd:audit lifeboat < path> as per itd-35 and owns only the chain sub-verb's contract. The two records disagree about which intent owns the lifeboat audit sub-verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 138.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030497817. Same register line against the same fact (itd-35 owns the lifeboat sub-verb); only the quoted second end differs (itd-16 itself rather than the intent chapter).
