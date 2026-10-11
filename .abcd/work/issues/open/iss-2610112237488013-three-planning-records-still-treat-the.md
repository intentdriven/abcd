---
schema_version: 1
id: "iss-2610112237488013"
slug: "three-planning-records-still-treat-the"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "lane for iss-2610111926302792 (abcd-0d), 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open/spc-42-every-change-starts-in-its-own-worktree.md"
remedy: "Waits on a product-thinker confirmation that the 2026-09-30 interview of itd-2609091014076309 discharges what spc-42, itd-148, iss-213 and iss-2608230847432285 wait on (spc-42 stays blocked on the store intent's build, not its interview). Once confirmed: rewrite spc-42's Blocked on to the store intent being built, name the check verb as an ahoy sub-verb per its Decision 5, update itd-148's notes and Open Questions, and restate iss-213's and iss-2608230847432285's deferral_reason through abcd capture defer, never by hand."
---

Three planning records still treat the worktree-store intent itd-2609091014076309's planning interview as owed, though it was held on 2026-09-30 (its record lists the rulings, and its Open Questions read 'None open'): spc-42's '## Blocked on' lists that interview as the blocker; iss-213's deferral_reason says 'Owed: the store draft's planning interview'; itd-148's Acceptance Criteria note and Open Questions defer to after it; and spc-42's Scope item 2 calls the check verb's name one of the store intent's open questions, though that intent's Decision 5 settled it as a sub-verb of ahoy. Found by the lane for iss-2610111926302792.

## Also 2026-10-12

A fourth record carries the same claim: iss-2608230847432285 (major, a sibling of iss-213) says in its deferral_reason "the store draft itd-2609091014076309 owns adding, listing and clearing ... itd-148 is blocked on that draft. Owed: the store draft's planning interview". Found by the ruthless review of the lane for iss-2610111926302792. A sweep of open issues, intents and specs on main found no other record owing that interview; .abcd/work/DECISIONS.md also mentions it but is append-only and stays as written.
