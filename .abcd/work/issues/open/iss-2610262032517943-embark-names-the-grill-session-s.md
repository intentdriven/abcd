---
schema_version: 1
id: "iss-2610262032517943"
slug: "embark-names-the-grill-session-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 327 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:3"
remedy: "Add glossary/core/embark.md for the shipped lifeboat-unpack command, and rename or qualify the grill-opening entry (glossary/interview/embark.md) so 'embark' no longer resolves to the unshipped concept; list core/embark in its not_to_be_confused_with."
finding_ref: cfk-982f17602f37b073
---

'embark' names the grill session's unshipped opening move in the interview glossary and the shipped lifeboat-unpack command in 03-embark

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:3` — "Point embark at a packed lifeboat and it writes that project's decisions, issues, intents, specs and release retrospectives into their canonical places in your repo"
- End B: `.abcd/development/brief/glossary/interview/embark.md:4` — "The opening move of a grill session, a design target that does not ship"

The glossary's only embark entry, interview/embark, defines embark as the opening move of a grill session, a design target that does not ship. 03-embark uses embark for the shipped /abcd:embark surface that unpacks a lifeboat into a repository. One name carries two unrelated concepts, and the shipped one has no glossary entry of its own, so a reader resolving the term through the glossary reaches the wrong concept.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 327.
