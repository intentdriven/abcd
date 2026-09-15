---
schema_version: 1
id: "iss-2609021457186209"
slug: "spc-2609021003136831-section-the-six-refusals-states-that-no"
severity: "critical"
category: "inconsistency"
source: "impl-review"
found_during: "itd-194 implementation, cold-reading Phase A"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary"
resolution: "The 21 glossary records under .abcd/development/brief/glossary/ now open with their frontmatter delimiter at line 0; the mattpocock/skills attribution comment moved to the line immediately after the block's closing delimiter, verbatim. reading assemble no longer refuses over this repository's own corpus at any assembling position."
impact: fix
---

spc-2609021003136831 section 'The six refusals' states that none of the six shapes appears in a committed record the include table admits, so the refusals refuse nothing on this corpus today. That measurement is wrong. Shape 2, the displaced frontmatter block, is carried by 21 committed glossary records under .abcd/development/brief/glossary/ (core/*.md, interview/*.md and _template.md), each opening with the attribution comment 'Adapted from mattpocock/skills (MIT). See README Acknowledgements.' as an HTML comment on line 0 and its frontmatter delimiter on line 1. The refusal is correct on the merits: frontmatter.Fields requires the delimiter at line 0, so those records' frontmatter keys are invisible to the assembler's redactor while every manifest asserts the key exclusions over them, which is the brief invariant 16 violation iss-2608301237456350 names. The consequence is that once itd-194 ships, reading assemble refuses at every assembling position over this repository's own corpus, and the ac-5 eval TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset, which assembles a clone of HEAD, cannot pass. The two spec requirements are not simultaneously satisfiable while those records stand. Measured remedy, verified on a scratch copy: moving the attribution comment to the line after the frontmatter block's closing delimiter in those 21 files makes the whole corpus assemble with no refusal and leaves record-lint with no new finding. Whether to repair the records, narrow shape 2's HTML-comment clause, or do something else is a corpus decision this spec does not authorise, so it is captured rather than taken.

## Grounds

- pursued: the repair is expected to make the whole corpus assemble at widening, entailment and detection with no refusal, keeping the attribution intact and record-lint, docs-lint and site-render clean; a fresh displaced-frontmatter refusal at any assembling position, a lost or altered attribution line, or a new record-lint finding over the glossary would show it wrong.
