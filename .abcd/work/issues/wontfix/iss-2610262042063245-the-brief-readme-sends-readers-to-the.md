---
schema_version: 1
id: "iss-2610262042063245"
slug: "the-brief-readme-sends-readers-to-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1046 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/README.md:13"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103568351]
finding_ref: cfk-1f9dab95fa252dd9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040339934. Identical A end and the same ruling that retires the word."
duplicates: [iss-2610262040339934]
---

The brief README sends readers to the roadmap dashboard for delivery state; the status-board intent retires the word and points it at the Now/Next/Later block.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/README.md:13` — "lives in the roadmap dashboard, never in brief prose"
- End B: `.abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:44` — "the word appears on neither surface; the glossary points `roadmap` at this block."

The brief README's delivery-state note says current delivery state lives in the roadmap dashboard. The shipped status-board intent rules that 'roadmap' appears on neither status surface and that the glossary points the word at the computed Now/Next/Later block on the board and the site's Status page. The README names a surface by a word the record has retired, and a reader following it would look for a roadmap dashboard.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1046.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040339934. Identical A end and the same ruling that retires the word.
