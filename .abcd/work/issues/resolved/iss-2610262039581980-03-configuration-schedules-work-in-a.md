---
schema_version: 1
id: "iss-2610262039581980"
slug: "03-configuration-schedules-work-in-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 900 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:428"
remedy: "Edit 03-configuration.md (lines near 428-430, 682, 1169, 1215): replace 'a later phase' with 'staged' or a dependency on the intent that would build it, per the retirement of phases (adr-2609212115255771)."
related_intents: [itd-2609211913453478]
finding_ref: cfk-ec59b609a78be04c
resolution: "03-configuration.md no longer schedules work in a later phase: schema versioning and migration (itd-9) and scheduled sync (itd-13) are called staged, each still naming its draft intent, and migrators are added if a later intent changes the shape (itd-9), as the dependencies page words the same fact. The fourth place the issue names (cross-project memory pages) already carried no phase and was left unchanged."
impact: internal
resolved_by:
  commit: "b2b450fed8e841706e525e743002e6ec10761419"
---

03-configuration schedules work 'in a later phase'; itd-2609211913453478 retired phases as the unit of sequencing.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:428` — "Schema versioning and cross-version migration come in a later phase"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

itd-2609211913453478 rules that phases and milestones are retired and that sequencing is dependencies plus the lifecycle shelves. 03-configuration still says schema versioning and migration come in a later phase, that migrators are added if a later phase changes the shape, that scheduled sync comes in a later phase, and that cross-project memory pages are a later phase. It uses the retired unit as if it still sequences work.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 900.

## Grounds

- pursued: the three places, in two sentences, at 03-configuration.md lines 896-898 and 2209 no longer say 'later phase' (count of later phase / later phases / later-phase on the page, any case and across line breaks: 3 at the base, 0 at the tip); each now says staged and names itd-9 or itd-13, and the migrator clause is a condition on a later intent with itd-9 cited, matching 02-constraints/02-dependencies.md; both intents sit under intents/drafts/, itd-9 scopes migrators, embark detection and the re-disembark refusal and changes no shape, and neither configuration schema migration nor scheduled sync ships; the cross-project memory pages passage (about line 1565) names no phase. It would be shown wrong if itd-9 or itd-13 were shipped, superseded or missing, if itd-9 were found to change a schema shape itself, if a configuration-record migrator or a scheduled dev-sync shipped, or if either sentence still sequenced work by phase.
