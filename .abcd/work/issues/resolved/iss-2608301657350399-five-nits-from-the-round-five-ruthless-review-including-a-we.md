---
schema_version: 1
id: "iss-2608301657350399"
slug: "five-nits-from-the-round-five-ruthless-review-including-a-we"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-round-5-ruthless"
found_at: "internal/core/grounds/grounds_test.go"
resolution: "All five fixed. Nit 1's guard is re-anchored on the property rather than the phrase: the refused set is proved to be exactly the runes below U+0020 in both directions, the message must name the serialiser it mirrors, and the over-claim ban is over the nouns that make a claim store-wide rather than one sentence. Nit 2's doc comment sits on requireGroundsFlag. Nit 3 gets its own groundsCheckExemption, asserted by string inequality with the claim exemption rather than by a banned word. Nit 4's test was deleted when grounds left frontmatter; the reader half is proved in capture and the gate half in lint, which is the split the nit asked for. Nit 5's comment carries the measurement — 3776 code points, 7 assigned, zero format characters and zero variation selectors — and the DECISIONS round-4 entry is corrected by an append."
impact: internal
resolved_by:
  intent: "itd-179"
---

five nits from the round five ruthless review including a weak negative assertion in the control refusal test

Five nits from the round-5 ruthless review, held on one record because each is
a sentence or a name rather than a behaviour.

1. `grounds_test.go` -- the guard added against the store-wide over-claim is
   bound to ONE spelling. Changing the refusal to say "no record field can
   CARRY" leaves the suite green and restores the defect
   iss-2608301646042379 records, verbatim in meaning. The test's second half
   (the four runes the store holds must not be refused) is substantive; only
   the negative substring is weak. This is the orchestrator's own test and the
   hit is fair.
2. `cli.go` -- `requireGroundsFlag`'s doc comment sits in the block that
   terminates at `func groundsUsageError`, so godoc attributes it to the wrong
   function and `requireGroundsFlag` has no doc at all.
3. `intent/ready.go` -- `groundsCheck` reuses `claimCheckExemption`, so the
   grounds row on a shipped record reads "a shipped record's CLAIMS are never
   backfilled". Wrong noun for the check reporting it; the same detail-string
   class as the resolved iss-2608300210588414.
4. `lint/schema_test.go` -- `TestCaptureReaderAcceptsGroundsKey`'s name and doc
   claim it proves the allow-list through the READER, but the body only calls
   `Lint` and never invokes capture. Package `lint` cannot import `capture`, so
   the name promises coverage the package cannot hold; the reader side is
   proved in `capture.TestReaderVerdictOnGroundsSpellings`.
5. `grounds.go` -- `isTextLetter`'s comment says the rest of the
   default-ignorable set is format characters and variation selectors. Measured:
   the table holds 3776 code points, of which 4 are Lo, 3 are Mn, and 3769 are
   unassigned; zero Cf and zero variation selectors, which live in the DERIVED
   property rather than this one. The conclusion the sentence supports is
   correct and was verified exhaustively; the description of the remainder is
   not. The same loose phrasing is in the DECISIONS.md round-4 entry.

## Grounds

- pursued: we expect a guard bound to one spelling to be defeated by the next rewording rather than by malice, and a mutation that swapped only the verb leaving the old assertion green while the new one fails on three counts is what shows the anchor moved
