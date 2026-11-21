---
schema_version: 1
id: "iss-2610262033108186"
slug: "itd-147-says-surface-chapter-prose-never"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 369 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:257"
remedy: "Reword the paragraph at 04-launch.md lines 257-262 so the prose names no flag or sub-verb and points at the generated appendix; the gate TestSurfaceChapterProseStatesNoShape matches only double-dash spellings, so the bare spelling passes it."
related_intents: [itd-147]
finding_ref: cfk-c218537f8277b705
refines: [iss-2609091956001547]
resolution: "The launch chapter's preview paragraph no longer states the preview's spelling or that it is a flag and not a sub-verb; it points at the generated appendix instead, and TestLaunchPreviewParagraphStatesNoShape holds the paragraph to that."
impact: internal
resolved_by:
  commit: "403e369598f111e428e910080daa2fd2469f0aea"
---

itd-147 says surface-chapter prose never states a flag; the launch chapter's prose names the `dry-run` flag

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:257` — "The preview is spelled `dry-run`, and it is a flag, not a sub-verb."
- End B: `.abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:90` — "the prose above it never states a flag or sub-verb."

itd-147 commits that each surface chapter's generated appendix carries the flags and sub-verbs and that the prose above it never states a flag or sub-verb, so a shape claim cannot drift. The launch chapter's prose says the preview is spelled `dry-run` and is a flag, not a sub-verb, and goes on to name `dry-run` as the preview's spelling several times. That is a hand-written shape claim of exactly the kind itd-147 says the prose no longer carries.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 369.

## Grounds

- pursued: the prose above the launch chapter's appendix states no preview spelling or shape in that paragraph; a return of `dry-run`, 'spelled', 'subcommand' or 'a flag, not a sub-verb' to it, or the loss of its pointer to the appendix, fails TestLaunchPreviewParagraphStatesNoShape
