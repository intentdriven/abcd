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
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Do reverses/duplicates/refines land on every record family at once, or on issues first through itd-2609212137116617?"
---

The decomposition discipline requires a cross-record link to be typed with one of four relations, supersedes or reverses or duplicates or refines, and never a vague related. Only the first of those four exists. supersedes is schema-known through recordHandleFields and is carried by fifty-eight decision records and three others, while reverses, duplicates and refines appear in no schema and in no committed record anywhere in the tree: the field list a record may carry is related_adrs, related_intents, builds_on and blocked_by, and a record carrying a key outside the known set is dropped by the reader rather than reported, so writing one of the three missing words would make the record invisible to every surface that reads it. The consequence is that an author following the rule literally either writes a link the tooling silently discards or writes the relation in prose and calls it typed, and the corpus shows the second: records assert a typed link in a heading and carry no frontmatter edge, which reads as done to a human and is invisible to every graph walker. That is the shape enforcement-claims-are-facts refuses, a convention naming a mechanism that does not exist, and it is load-bearing here because the decomposition protocol is the documented gate until its automated rung ships. Fix candidates, none chosen: implement the three missing relations as known fields so the rule can be followed; or narrow the rule to the vocabulary the schema implements and say what an author does with a relation the four words cannot express; or record that prose is the sanctioned form for the three and stop calling them typed. Detector: a record asserting a typed relation in its body carries a frontmatter edge naming the same target, and a relation the schema cannot express is refused at write time rather than dropped in silence.
