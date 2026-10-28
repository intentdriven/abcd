---
schema_version: 1
id: "iss-2610262032516143"
slug: "embark-s-lessons-speak-of-the-previous"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 326 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:57"
remedy: "none (filed automatically)"
finding_ref: cfk-a1655910dea6bf42
---

embark's lessons speak of 'the previous voyage' and 'the new voyage's brief'; the voyage glossary restricts the word to the operations namespace and forbids it for a project

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:57` — "answer what the previous voyage learned (itd-24)"
- End B: `.abcd/development/brief/glossary/core/voyage.md:59` — "Do not use it for a "project" (too generic)"

core/voyage fixes voyage as the operations namespace that records disembark and embark runs, says not to use it for a project, and leaves the end-to-end project-lifecycle sense unnamed after adr-35. 03-embark's lessons sub-verb answers what the previous voyage learned and ranks lessons against the new voyage's brief, using voyage for a project's lifecycle, the retired sense.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 326.
