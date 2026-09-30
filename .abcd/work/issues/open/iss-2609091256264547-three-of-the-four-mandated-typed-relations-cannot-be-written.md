---
schema_version: 1
id: "iss-2609091256264547"
slug: "three-of-the-four-mandated-typed-relations-cannot-be-written"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/principles/decompose-before-filing.md"
remedy: "Waits on the remainder's planning question (every family at once, or family by family): add reverses beside duplicates and refines in issueschema.Known, the issuerecord validator and lint.recordRefFields, then carry all three to intent and ADR frontmatter; if at once, in one change with a round-trip test per family; if family by family, issues first, then intents, then ADRs. Prove it with a record-lint rule refusing a body that asserts a typed relation with no matching frontmatter edge."
deferred_after: v0.11.1
deferral_reason: "The product thinker's ruling M23 of 2026-09-23: support all four relations as typed fields on every record and every reader. Partly delivered: the filing-time match (itd-2609212137116617, shipped) writes duplicates and refines, and the lint resolves them; reverses exists in no schema, and the other families have not been planned. Owed: planning the remainder, which opens on one question: do the missing relations land on every record family at once, or family by family?"
---

The decomposition discipline requires a cross-record link to be typed with one of four relations, supersedes or reverses or duplicates or refines, and never a vague related. Only the first of those four exists. supersedes is schema-known through recordHandleFields and is carried by fifty-eight decision records and three others, while reverses, duplicates and refines appear in no schema and in no committed record anywhere in the tree: the field list a record may carry is related_adrs, related_intents, builds_on and blocked_by, and a record carrying a key outside the known set is dropped by the reader rather than reported, so writing one of the three missing words would make the record invisible to every surface that reads it. The consequence is that an author following the rule literally either writes a link the tooling silently discards or writes the relation in prose and calls it typed, and the corpus shows the second: records assert a typed link in a heading and carry no frontmatter edge, which reads as done to a human and is invisible to every graph walker. That is the shape enforcement-claims-are-facts refuses, a convention naming a mechanism that does not exist, and it is load-bearing here because the decomposition protocol is the documented gate until its automated rung ships. Fix candidates, none chosen: implement the three missing relations as known fields so the rule can be followed; or narrow the rule to the vocabulary the schema implements and say what an author does with a relation the four words cannot express; or record that prose is the sanctioned form for the three and stop calling them typed. Detector: a record asserting a typed relation in its body carries a frontmatter edge naming the same target, and a relation the schema cannot express is refused at write time rather than dropped in silence.

## Deferral 2026-09-29

Deferred past v0.11.1: The product thinker's ruling M23 of 2026-09-23: support all four relations as typed fields on every record and every reader. Partly delivered: the filing-time match (itd-2609212137116617, shipped) writes duplicates and refines, and the lint resolves them; reverses exists in no schema, and the other families have not been planned. Owed: planning the remainder, which opens on one question: do the missing relations land on every record family at once, or family by family?

## Remedy grounds (2026-09-29)

- Ruling M23 (2026-09-23, .abcd/work/DECISIONS.md) chose to build all four relations; duplicates and refines are schema-known on issues (internal/core/issueschema/issueschema.go), and reverses appears in no schema.
- The body-edge lint is the record's own detector and stops the prose-called-typed shape from recurring.
- Rejected: narrowing the rule to the implemented words, which the ruling declined.
