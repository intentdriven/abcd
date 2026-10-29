---
schema_version: 1
id: "iss-2610262032548523"
slug: "itd-10-lists-a-lifeboat-among-the-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 334 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-10-purge-uninstall.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-10]
finding_ref: cfk-455a5a933ded9090
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031329436. Identical A end and the same out-of-tree lifeboat fact, cited from the disembark chapter instead of the platform chapter."
duplicates: [iss-2610262031329436]
---

itd-10 lists a lifeboat among the .abcd/ contents destroy deletes; disembark always writes the lifeboat out-of-tree

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-10-purge-uninstall.md:16` — "AND the entire `.abcd/` namespace (including lifeboat, dev artefacts, logbook, and config)"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:198` — "The lifeboat is written out-of-tree, so the source repo has nothing to gitignore."

itd-10's destroy promises to delete the .abcd/ namespace 'including lifeboat'. The disembark chapter writes every lifeboat to an operator-chosen destination outside the source repository, so the source has nothing to gitignore. That leaves no lifeboat under .abcd/ for destroy to remove, and the draft's list of what it deletes names an artefact the surface never puts there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 334.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031329436. Identical A end and the same out-of-tree lifeboat fact, cited from the disembark chapter instead of the platform chapter.
