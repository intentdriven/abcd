---
schema_version: 1
id: "iss-2608301836222858"
slug: "grounds-vocabulary-is-read-by-no-surface"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-delta-close"
found_at: "internal/core/grounds/grounds.go"
resolution: "The vocabulary is now rendered rather than copied. The one copy that was a GATE, the degeneracy set, is built from Vocabulary; the six copies a user reads are rendered by grounds.UsageSpelling and grounds.ProseList, and the refusals keep rendering through vocabularyList. The rendered strings are byte-identical, so the generated CLI reference page is unchanged. What stays true, deliberately: the prose documentation still spells the values by hand, because prose is free text and the maintainer scoped it out; and a test now pins the three renderings, so a change to the closed set fails beside the set rather than in a regenerated page nobody read."
impact: internal
resolved_by:
  intent: "itd-179"
---

grounds.Vocabulary claimed to be the one copy every gate and every flag description reads, and no surface reads it

Found by an unused-export sweep while settling the itd-179 delta nits, not by
any of the delta reviews.

`grounds.Vocabulary` carried the comment "It is the ONE copy every gate and
every flag description reads." Nothing outside package `grounds` reads it. The
closed set is rendered for refusals by the package-internal `vocabularyList`,
and every surface spells the three values as literal text instead: four sites
in `internal/surface/cli/cli.go`, one in `internal/core/capture/grounds.go`,
plus `commands/intent.md` and the generated `docs/reference/cli/commands.md`.

The comment is corrected in the change that found it, so the claim now matches
the code. What stays open is the consolidation the claim described.

It is a nitpick because the literals and the closed set agree today. The cost
is what a fourth value would take: five edits in code plus the plugin surface
and a regenerated reference page, which is the arithmetic a closed set exists
to avoid.

`grounds.Heading` has the same shape one step smaller — the `## Grounds`
spelling is a literal in record-lint's remedy string because `core/lint` does
not import `core/grounds`. That one is left alone; its comment is narrowed to
what it establishes rather than an import being added for one prose string.

Amended 2026-08-30 after the fix-delta review: the enumeration above MISSES a
copy, and it is the one that matters most. `degenerateWords`, in the same file
eighty lines below `Vocabulary`, spells the three tokens again and its own doc
calls it "the vocabulary itself".

Every other copy is a MESSAGE -- a flag description, a usage string, a docs
page. This one is a GATE. Add a fourth token to `Vocabulary` and forget this
map, and `ValidateText` silently stops refusing a grounds text made solely of
the new token, while the refusal it declines to raise still says the text only
repeats the vocabulary. So the count is not five edits in code but six, and this
is the one to consolidate first.

Amended again 2026-08-30, and this is the amendment that says stop amending.

The previous amendment claimed the enumeration missed "a copy" and put the count
at six code sites. It is at least SEVEN, plus two doc sites nobody listed:
`internal/core/intent/ready.go`'s `groundsRemedy` -- live, printed to the
operator by the readiness gate, and whose own doc calls itself "the one spelling
of how a ground is recorded" while four `cli.go` copies falsify that --
`commands/capture.md`, and `internal/README.md`. A tree sweep finds twelve files
carrying a literal spelling.

**Do not amend this list again by hand.** Three consecutive corrections have each
introduced a fresh error, because a prose enumeration of copies cannot fail and
so cannot be maintained. The remedy is iss-2608301918362294: make it executable,
and decide there what counts as a copy, since the package that defines the tokens
legitimately spells them.

## Grounds

- pursued: we expect a rendered vocabulary to make its copies unable to disagree, where four hand-corrected enumerations of them each introduced a fresh error
