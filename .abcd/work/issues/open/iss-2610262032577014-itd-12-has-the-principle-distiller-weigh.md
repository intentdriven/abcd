---
schema_version: 1
id: "iss-2610262032577014"
slug: "itd-12-has-the-principle-distiller-weigh"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 340 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-12-work-adapter-weighting.md:15"
remedy: "Re-scope itd-12 to the inputs the shipped principle-distiller reads (ADRs, intents, resolved issues and graveyard findings in a packed lifeboat), or record that it is blocked on the pack carrying notes, reviews and memory and on dev-sync (staged); today .abcd/work/notes/ does not exist"
related_intents: [itd-12]
finding_ref: cfk-d20d64374dcf128f
---

itd-12 has the principle distiller weigh .abcd/work/notes/, reviews and memory; disembark's agents read only packed lifeboat files, and the pack carries none of those sources

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-12-work-adapter-weighting.md:15` — "abcd weights `.abcd/work/notes/` (shared working scratch) below ADRs, specs, and memory in the principle distiller"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:307` — "The four lifeboat agents read the packed lifeboat's files"

itd-12 ranks ADRs, specs, memory, reviews and notes as the principle distiller's inputs. The disembark chapter says its lifeboat agents, principle-distiller among them, read the packed lifeboat's files. The pack holds the brief, graveyard, intents, specs, retrospectives, ADRs and an issue snapshot, but no notes, reviews or memory. As the pack is written, the weighting the draft depends on has nothing to weigh.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 340.
