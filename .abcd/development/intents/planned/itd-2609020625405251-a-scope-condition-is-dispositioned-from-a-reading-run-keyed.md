---
id: itd-2609020625405251
slug: a-scope-condition-is-dispositioned-from-a-reading-run-keyed
spec_id: spc-2609020626046252
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-181, itd-177, itd-180, itd-185]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A scope condition is dispositioned from a reading run, keyed to the condition's identity and joined to the item that occasioned it

Typed links: `builds_on` [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) (scope-condition disposition at verdict ingest), [itd-177](../shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md) (condition identity), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (superseding dispositions), [itd-185](../shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md) (the ingest that mints items); `refines` [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) (a second writer into the same surface).

## Press Release

> **A detection can change the standing of an assumption, on the record.** `abcd intent condition <itd-N> <cond-id> --disposition <survived|narrowed|falsified|untested> --occasioned-by <rdi-N|itd-N> --grounds "<why>"` writes one scope-condition disposition against a shipped intent, keyed to the condition's stamped identity, joined to the reading item, or the shipped intent, that occasioned it, with the narrowing stated where the value is narrowed. Until now only the fidelity verdict could disposition a condition; now a reading or a delivery can occasion one, and the record shows which did.

> "When the detection pass tells me a condition I assumed does not hold in the tree, I want to record that against the condition, not against the sentence, and I want the item that told me to be named," said an AI/agent researcher who runs the detection pass over their own shipped state.

## Why This Matters

The cold-reading design has, at Step 6 in Iteration 2, each scope condition dispositioned by the researcher against what the detection pass returned, and it makes the disposition warm: never passed to a reading, populated at Step 6, and in Iteration 1 exercised by the fidelity verdict only. [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) delivered that surface: the enum, the identity key, the narrowing rule, and one writer, `abcd intent audit ingest`, which takes the auditor's verdict. There is no path from a reading run to a condition. A detection item names a tension and the constraint in play, and the researcher's response to it is a disposition on the item; but the condition the item bears on stays as the fidelity verdict left it, or unstamped, and the record cannot show that a reading changed an assumption's standing.

The join matters for the closing run. The design's purpose-durability reading turns on a tension rejected with a named purpose, over a state deliberately unchanged, returning again; a condition that was narrowed because of a detection is the clearest form of a state deliberately changed.

## What's In Scope

- **`abcd intent condition`** as a second writer into the same disposition surface the verdict ingest writes, keyed to the condition identity, refusing a value outside the enum, a narrowing on any value but narrowed, a narrowed value without a narrowing, an occasion that does not resolve to a reading item or to a shipped intent, a ground below the substance floor, and an intent not in `shipped/`.
- **How the two writers coexist.** The section holds one dated block per write. A verdict ingest writes one block covering every condition, as it does today; this verb writes one block covering one condition and naming its occasion. A condition's standing disposition is the one in the latest block that names it, and the render says which block it came from.
- **The occasion recorded** with the disposition, so a condition's standing carries the item, or the delivery, that changed it.
- **Exclusion from readings** unchanged: dispositions render under the heading the assembler already withholds.

## What's Out of Scope

- The reading marking conditions itself. The reading names tensions; the researcher marks, on the rule itd-181 adopted. This is a registered interpretation, because the design framework's letter reads the other way: Section 7.1 has the reading mark each condition survived, narrowed, falsified or untested, and section 14 has each scope condition dispositioned under Iteration 2's cold column. The ground for reading both as the reading naming the condition and the researcher marking it is that the readings companion's ratified registrative body (section 4.2) carries tension, constraint in play and why it is a tension, with no field for a mark; that the companion (section 9) says readings do not disposition their own items; and that the build sheet's W8 makes dispositions warm and never passed to a reading. The join from the researcher's mark to the item is by citation: Where the constraint in play is a scope condition, the detection item cites the condition's identity, and the spec states how the verb reads that citation.
- Dispositioning a condition on a draft or planned intent. Conditions are dispositioned on shipped intents, where there is a delivered state to disposition against.

## Mechanism

We expect a writer keyed to condition identity and joined to a reading item to make a reading's effect on the frame's assumptions countable because the identity survives rewording and the join names the cause, so the closing run can ask which conditions a reading changed. It fails if the verdict ingest and this verb write incompatible shapes, which sharing one validator and one block grammar prevents.

## Scope Conditions

- The verb writes against a shipped intent only. A condition on a planned intent has no delivered state and is out of scope. <!-- cond: cond-2609020626040782 -->
- The occasion is a reading item at any position, not only detection, because an entailment item can bear on a context claim as readily as a detection can; or a shipped intent, because a delivery can change a condition's standing as a reading can, which is how the comparative refusal's surviving condition on itd-199 is re-dispositioned when the comparative channel ships. <!-- cond: cond-2609020626040385 -->
- Dispatch from a reading item to the conditions it occasioned lands with the record dispatcher's coverage of reading items, which is planned work; until then the join is read from the intent's side. <!-- cond: cond-2609020626040303 -->

## Acceptance Criteria

- **Given** a shipped intent with a stamped condition and a reading item, **when** the verb runs with `falsified` and a ground, **then** the condition carries that disposition, the occasion and the ground, and the intent's rendered notes show it in its own dated block.
- **Given** the verb with `narrowed` and no narrowing, **when** it runs, **then** it refuses and names the missing narrowing.
- **Given** the verb with `survived` and a narrowing, **when** it runs, **then** it refuses.
- **Given** a value outside the four, **when** it runs, **then** it refuses and names the enum.
- **Given** an occasion that does not resolve, **when** it runs, **then** it refuses.
- **Given** a planned intent, **when** the verb runs, **then** it refuses and names the bucket.
- **Given** a condition already dispositioned by the verdict ingest, **when** the verb writes a second disposition, **then** both blocks stand and the later one is reported as standing.
- **Given** a reading assembled over the intent, **when** the manifest is read, **then** the condition dispositions are asserted excluded and none reaches the bundle.

## Prior Art

- [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) and its spec; [itd-177](../shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md) (condition identity); [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (superseding dispositions).
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a second writer keyed to condition identity and joined to a reading item to make a reading's effect on the frame's assumptions countable for the closing run; a condition changed because of a detection that the record cannot trace to that detection would show it wrong
