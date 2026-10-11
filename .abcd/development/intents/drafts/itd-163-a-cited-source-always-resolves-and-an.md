---
id: itd-163
slug: a-cited-source-always-resolves-and-an
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
supersedes: [itd-145]
builds_on: []
severity: minor
impact: additive
---

# A cited source always resolves and an adopted inspiration is always acknowledged: the reference-closure and acknowledgements-mirror gate

Typed links: **supersedes itd-145** (the acknowledgement convention
arming itself; maintainer-ruled 2026-08-28). itd-145 was the seed
ambition — any record citing an external source without its entry fails;
this intent delivers its mechanically checkable core (closure and
mirroring over the declared sets) and deliberately does not attempt the
seed's harder residue, detecting a prose-named source that never enters
the citation apparatus. That residue survives only as the open question
below about the registry's `kind: tool` entry class.

## Press Release

> **abcd's credit surfaces close deterministically: a lint family proves, offline and on every push, that each citation the durable record uses resolves to an entry in the CSL references, that the CSL references and the acknowledgements References section mirror each other exactly, and that every influence declared in a committed influence registry carries its Inspirations entry.** The acknowledgements convention — an entry lands in the same change that adopts a pattern, cites a source, or integrates a tool — is a promise the file's own header makes, and until now it was kept by discipline alone. Discipline held for one era and lapsed for another; the gate turns the promise into a refusal. The rules extend the existing citation lint family rather than standing beside it, and like the rest of that family they read only committed files: the gate never dials out.

> "I trust the gates in this repository precisely because they refuse instead of advising," said Bob, a staff engineer who reviews the record before they rely on it. "The acknowledgements file told me it grows with the work. What I found was that the most formative influences were credited everywhere except there. A mirror that is checked is a mirror I can cite; a mirror kept by hand is a hope."

## Why This Matters

The 2026-08-28 attribution review found no licence violation anywhere in the tree, and one failure mode everywhere it found anything: a design-shaping influence credited fully in the record tier (intents, related-work, the brief) that never propagated to `ACKNOWLEDGEMENTS.md`, the surface the project points the public at. The references section and `references.csl.json` match 1:1 today only because a hand kept them so. The repository's own doctrine says what to do with a convention that fails silently: arm a detector, and make it refuse rather than warn.

Closure is also what makes the credit surface legible to an outsider. A reader who finds a source in the record and looks for it in the acknowledgements must find it or find nothing missing; a reader who finds an Inspirations entry must be able to trace it back to the registry entry that declares where it is used. The gate makes both directions checkable facts.

## What's In Scope

- A reference-closure rule: every citation key or reference-style source link the durable record uses resolves to an entry in `references.csl.json`.
- An acknowledgements-mirror rule: the CSL file and the acknowledgements References section list the same entries, both directions; a committed influence registry and the acknowledgements Inspirations section likewise mirror, both directions.
- The influence registry itself as committed data: one entry per external influence with title, author, source, licence field, and the record id (intent, ADR, or issue) where the adoption lives.
- Extension of the existing citation lint family (the spc-17 rules): same config surface, same zero-network design bet, same exit-code convention.
- Day-one green: the gate arms in the same change that lands the acknowledgements backfill (iss-2608280824478819), so the first armed run passes on a tree that has just been made honest.

## What's Out of Scope

- **Licence vetting.** Whether a registry or CSL entry's licence field is populated and current is itd-164's gate; this intent only carries the field.
- **Network access of any kind.** The rules read committed files; anything live belongs to the refresh verb (itd-164).
- **Mechanical detection of unregistered influences.** The gate proves the declared set is mirrored; it cannot prove an undeclared influence exists. Admission to the registry stays a human act, per the review-then-record protocol.
- **Rewriting history.** The gate judges the tree, not past commits.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a durable-record page citing a key absent from `references.csl.json`, **when** the gate runs, **then** it refuses, naming the key and the file.
- **Given** a CSL entry with no matching acknowledgements References line, or a References line with no CSL entry, **when** the gate runs, **then** it refuses, naming the unmirrored entry and the direction.
- **Given** an influence registry entry with no Inspirations entry, or an Inspirations entry absent from the registry, **when** the gate runs, **then** it refuses symmetrically.
- **Given** the backfilled tree, **when** the gate is first armed, **then** it reports zero violations, and the arming change and the backfill are the same change.
- **Given** any evaluation of these rules, **when** they run, **then** no network access occurs; the rules read only committed files.

## Open Questions

- Registry format: one JSON file beside `references.csl.json`, or one frontmattered markdown file per influence? The JSON form is easier to mirror-check; the per-file form matches how the rest of the record is kept.
- Does the Inspirations section become generated from the registry (single source, like the changelog) or merely checked against it? Generation removes drift permanently but changes how contributors edit credit.
- Closure scope: the spc-17 rules deliberately scope citations to footnote definitions in docs; the closure rule needs a stated scope for the development tier (intents, ADRs, principles), where citations are reference-style links rather than footnotes.
- The "integrates a tool" clause of the acknowledgements header has no mechanical trigger; does the registry grow a `kind: tool` entry class so integrations are at least declarable, with detection left to review?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
