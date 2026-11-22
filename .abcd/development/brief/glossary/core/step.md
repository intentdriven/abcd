---
term: step
bounded_context: core
definition: One of the ordered, independently landable pieces a spec lists under its Steps section; each step is one lane and one pull request, and a spec with no steps is one step.
aliases: []
forbidden_synonyms: ["task", "sub-task", "scope"]
status: stable
introduced_in: adr-2609212115255771
starts_when: null
ends_when: null
not_to_be_confused_with: core/spec
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# step

A **step** is the unit below a spec (itd-2609212103565953, planned): the spec's author lists them in order, each with its own footprint, and the build loop (`abcd build`) takes them one lane at a time, the spec's next pending step opening the next lane when a lane is done. A step that does not fit a cut leaves the rest as the spec's remainder. In the build loop the word names the spec's piece and nothing else (BU1, iss-2609291313276243): the build loop lands each step as one lane and takes that lane through its **stages** — worktree, brief, implement, validate, land — and `abcd implement step` performs one stage, not one step. The run's state file and the loop's payloads keep the two apart: `spec_step` and `step_title` name the spec's step, `stage` the lane's.

## Senses

The word carries two senses in the record, and the glossary fixes one spelling for each
(iss-2610262038278710).

| Sense | The one spelling | Where it lives |
|---|---|---|
| One of the ordered, independently landable pieces a spec lists under its Steps section | **step**, unqualified, or "a spec's step" where a land step is also in view | the spec's own Steps section, and the build loop's `spec_step` and `step_title` |
| One act of the land verb's own sequence, `origin`, `gate`, `announce`, `withdraw`, `push` and `enqueue`, or `land` alone, each reported as an entry of the JSON document's `steps` field with its `status`; "every step is idempotent" means each such act, run again, resumes where the last run stopped | **a land step**, or bare "step" only inside the land chapter's own "The steps" section and the `steps` field | [`04-surfaces/38-land.md`](../../04-surfaces/38-land.md), The steps and the JSON document |

**A land step is not a spec's step.** A spec's step is a piece of the design record that one
lane lands; a land step is one act inside a single landing of one branch, and a landing runs
all of its land steps for whatever branch it is given, a spec's step or not. The land chapter
uses both senses on one page: its own acts in "The steps" and the `steps` field, and a spec's
step where it cites one (spc-2610150728396006 step 4). Nor is a land step a build-loop
**stage**: a stage is one of the five a lane passes through, and the land verb's steps belong
to the verb a person runs on a branch.

## When to use

When a spec is larger than one implementer can hold and land. Not as a task tracker: a step is a section of the design record, not a record of its own.

## Related terms

- [record families](record-families.md) — the one page that maps the record families and how they relate
