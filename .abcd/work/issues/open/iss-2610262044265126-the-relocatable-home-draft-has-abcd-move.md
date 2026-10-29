---
schema_version: 1
id: "iss-2610262044265126"
slug: "the-relocatable-home-draft-has-abcd-move"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1206 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does abcd offer a person-chosen location for its user-level home (itd-77), or is the home fixed at the .noindex folder under the person's home directory? Answers: (a) withdraw itd-77, since the shipped rename ruling (itd-2610030720038073) fixed the name, dropped all move code and relies on the folder name for indexer exclusion, and the home resolver hard-codes the name; (b) keep it and re-scope to an explicit person-run relocation that keeps a .noindex-named folder and re-points the trust readers. Recommended (a): a relocated home loses the name the indexer exclusion rests on and the trust readers refuse a symlinked home, so the feature cannot be built without reopening a ruling made on 2026-10-03; capture afresh if still wanted."
related_intents: [itd-77, itd-2610030720038073]
finding_ref: cfk-4342c3d4c83679f1
---

The relocatable-home draft has abcd move its home on command, but the shipped rename ruled that abcd moves nothing and fixed the name at ~/.abcd.noindex

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17` — "a guided, wizard-style flow moves the home wherever the user's filing system wants it"
- End B: `.abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md:99` — "abcd moves nothing. An existing `~/.abcd` stops every abcd command and hook before it writes anything"

itd-77 promises a guided flow in which abcd moves the user-level home anywhere and re-points every consumer. itd-2610030720038073 dropped abcd's own move of the home ('abcd moves nothing') and carries no move code. It also relies on the `.noindex` name for the indexer to skip the folder, which a home moved by name elsewhere would lose. The two records disagree on whether abcd moves its home.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1206.
