---
id: itd-197
slug: a-fact-a-source-can-settle-is-read-not-recalled
spec_id: null
kind: discipline
kind_notes: "A rule every assertion inherits: recall locates a source, it never quotes one."
suggested_kind: discipline
reclassification_history: []
builds_on: [itd-195]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A fact a source can settle is read, not recalled

## The rule

Before stating what a record, a rule, a spec, a command or a file says, open it
and read it. **Recall is for locating the source. It is never for quoting one.**

The bound matters, because "verify everything" is paralysis rather than rigour.
The rule binds an assertion that is about to be **written into a record, a commit
message or a brief, or used as the grounds for a decision**. Conversation may be
provisional, and the honest form there is to say which it is: "the intent says X"
and "I recall the intent saying X" are different sentences and only one of them
survives being wrong.

## Why

Measured, in one session, four times. Each was a document the author had read
carefully hours earlier, and each was stated with confidence.

- The grounds put to the maintainer for degrading a gate said the design licensed
  it "when a signature proved noisy in practice". The design's actual condition
  is *degradation only on observed noise*, and the evidence in hand was a
  constructed corpus over an instrument that has never run. The maintainer caught
  it by asking whether the claim was really true.
- A captured issue asserted that no non-test file in a package referenced the
  agent definitions. One already declared the directory constant and read it. The
  builder that the capture briefed caught it.
- A residue paragraph claimed an enumeration "cannot fall behind the surface it
  guards". Two written lists survived inside it, and the guard's own header said
  so in capitals.
- A synthetic battery's result was reported as "measured: they do not", of an
  instrument no reading had ever been dispatched to.

The pattern is the finding. **Confidence tracked familiarity, not accuracy.**
Every error was about material read recently and attentively; recency produced
confidence, and confidence displaced the check. The documents nobody had looked
at that day produced no false claims at all, because nobody felt able to assert
anything about them.

The cost is asymmetric to the point of being decisive. Opening the file costs
seconds. A recalled claim enters a ledger record, briefs an agent, or becomes the
premise of a maintainer's decision, and is then acted on by people who cannot
tell recall from reading because the sentence looks identical either way.

## Why this is not a memory item

**It already was one, and that is the evidence.** The session that produced the
four errors above carried an agent-memory note from a previous session saying, in
substance, to check a principle's letter before citing it. It was in context from
the first turn. It did not bind.

A rule enforced only by an agent remembering to apply it fails precisely when the
agent is confident, which is exactly when it is needed. That is the promotion
signal
[memory-graduates-to-record](../../principles/memory-graduates-to-record.md)
describes, in its stronger form: not merely a lesson recalled twice, but one
recalled and then not followed, which shows the home rather than the lesson to be
at fault.

## The gate

- **Given** a statement about what a record, rule, spec, command or file says,
  **when** it is written into a record, a commit message or a brief, or used as
  grounds for a decision, **then** the source was opened for it, or the statement
  is not made.
- **Given** a fact stated from recall in conversation, **when** it is stated,
  **then** it is marked as recalled rather than checked, so a reader can tell the
  difference before relying on it.
- **Given** such a claim found to be wrong, **when** it is corrected, **then** the
  correction says whether the original was read or recalled, because a corpus of
  those is what would justify automating any part of this.

## Fit

[itd-195](itd-195-a-claim-about-how-the-code-behaves-is-executable-or-it-is-no.md)
is the complement and neither contains the other. That rule governs a claim about
how code BEHAVES, and its remedy is a test. This one governs a claim about what a
source SAYS, and its remedy is to read it. A quotation cannot be made executable
and a behaviour cannot be settled by quotation, so the two are disjoint halves of
one instinct: do not write what you have not checked.

## Staging

**Adopted 2026-08-31 at the documented-protocol rung.** No detector, and it is
not obvious one is possible: distinguishing a recalled assertion from a read one
is not visible in the text, which is the whole difficulty. What could be
mechanised later is narrower, a check that a record citing another record by id
quotes text that is actually in it.

The existing corpus does not conform and is not assumed to. The four errors above
were corrected; nothing else has been surveyed.
