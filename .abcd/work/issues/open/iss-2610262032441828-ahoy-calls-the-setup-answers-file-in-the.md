---
schema_version: 1
id: "iss-2610262032441828"
slug: "ahoy-calls-the-setup-answers-file-in-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 310 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:874"
remedy: "In 01-ahoy.md call the setup answers 'the answers file' (and the provenance and path-entry artefacts 'files') where no id or minting command makes them records, as glossary/core/record.md reserves the word; the uses are near lines 819, 961 and 973."
finding_ref: cfk-d2fa072897f6c63c
---

ahoy calls the setup answers file in the local tier an 'answers record'; the record entry says an artefact with no id and no minting command is not a record

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:874` — "answers file write one answers record through `interview.Write`"
- End B: `.abcd/development/brief/glossary/core/record.md:57` — "an artefact with no id and no minting command is neither"

core/record reserves 'record' for an identified, filed document a command mints and a lint gate reads, and says a working-tier note or a local scratch file, having no id and no minting command, is not one. 01-ahoy calls the setup answers written into .abcd/.work.local/interviews/ an answers record, and its path-entry and provenance files records too, though none carries an id.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 310.
