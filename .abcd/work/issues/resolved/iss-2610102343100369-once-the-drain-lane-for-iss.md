---
schema_version: 1
id: "iss-2610102343100369"
slug: "once-the-drain-lane-for-iss"
severity: "nitpick"
category: "drift"
source: "review-followup"
found_during: "abcd-60 drain run 2026-10-11, lane for iss-2610072347247487"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat"
remedy: "Update the personas row's expected status in the lifeboat mapping Table and regenerate the brief's 00-meta.md mapping table, so the M0 prediction matches the probe (grounded from an authored brief file, partial from a stub, blank with none); or record why the prediction is kept as written."
resolution: "The personas row of lifeboat.Table now predicts grounded at abcd-native (blank below), matching the native personas adapter. The package doc and the brief's thesis bullet say so, and the brief's 00-meta.md mapping block was re-rendered from renderMapping()."
impact: internal
resolved_by:
  commit: "f516be2b8fcff88d02a04bf84a07039b66ffa8df"
---

Once the drain lane for iss-2610072347247487 lands, the lifeboat probe grounds the personas section from an authored brief file, as it does every other human-owned section. The design record still predicts the old ceiling: the personas row of the lifeboat mapping Table (internal/core/lifeboat mapping.go, around line 151) and the generated mapping table in the brief's 00-meta.md say 'only partial there'. The probe now exceeds that on abcd's own tree, which has about 859 bytes of personas prose. Both of that lane's reviewers noted it as outside its remedy.

## Grounds

- pursued: TestPersonasPredictionMatchesTheProbe keeps the row's abcd-native prediction equal to the best status the native personas adapter reaches (authored file grounded, stub partial, none blank), and checks no git-tier or conventions-tier source reads personas. It would be shown wrong if the adapter's ceiling changes, or if a poorer-tier personas adapter is added, without the row following.
