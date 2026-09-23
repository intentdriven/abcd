# `/abcd:identity` — Repo Positioning

A project's tagline gets written once and copied four times: the README
strapline, the plugin manifest, the conventions file, a string baked into the
binary's banner. Then one of them is improved. `/abcd:identity` records the
canonical wording in one place, tells the maintainer which surfaces have drifted
away from it, and prints the exact diff that would bring each back.

The bare form and the rendered diff are **strictly read-only**. Initialisation
is the single write path, and it runs once, at onboarding.

It answers a different question from `/abcd:lint`. `lint` reports whether the
repo conforms to the working conventions as a whole and runs the positioning
check as one rule among them; `identity` is where a maintainer looks at the canon
itself and at what a fix would read like.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `init` | — | shipped |
| `render` | audit | shipped |

## The identity block

The canonical home is a markdown block in the repo's own record. Markdown stays
the single source of truth, and the committed configuration records only where
the block lives:

```markdown
## Identity (canonical)

- **Title:** abcd — Agent-Based Configuration for Development
- **Tagline:** For people who know what they want to build and need help shipping it.
- **Pitch:** A single Go binary that carries the why from idea to shipped
  reality, usable as a plugin in compatible agent harnesses.
```

Title and tagline are required; the pitch is optional at onboarding and may wrap
across lines. abcd's own block is the "Identity (canonical)" section of
[`01-product/README.md`](../01-product/README.md).

## What a repo registers

`.abcd/positioning.json` records the block's location, the family severity, and
the surfaces held to it. It sits beside the repo's other per-concern
configuration rather than under `.abcd/config/`, where `identity.json` already
means the git commit-author pin.

The severity is the whole family's weight, and it is the one knob that changes
what a failed check costs. `warn`, the default, reports drift as a warning that
does not fail the run. `blocker` promotes every positioning finding to an error,
so a repo that wants its own strapline treated as load-bearing turns the
advisory family into a hard `abcd lint` gate with one word.

A registered surface names candidate files (the first that exists is checked, so
one entry covers several manifest formats), how to locate the text inside one (a
regular expression with exactly one capture group around the text to compare, or
a top-level JSON field), which block fields it requires, and the template a
proposal renders from. The single group is a rule, not a convention: a pattern
carrying two makes the whole registry invalid and the check refuses to run,
naming the pattern and the count it found. Leaving the surface list
empty adopts the canonical three — the README strapline, the plugin manifest
description, and the conventions-file opening; naming any replaces them, so
nothing is ever registered silently.

The registry is all-or-nothing: it is decoded with unknown fields disallowed and
validated in full before any of it is used, because every byte of it arrives as
committed data. `schema_version` is required and must be `1`. An unloadable
registry is reported as a warn-tier finding rather than as drift, so the symptom
reads as a broken check rather than as a missing tagline.

## Behaviour

Bare, the verb prints the block and every surface's verdict, read-only. Its
render prints a unified diff per drifted surface and writes nothing. Its
initialiser records the block and the pointer to it.

Comparison is by normalised containment: markup, dashes, line wrapping, and case
are folded away, so a tagline bolded mid-sentence or wrapped across two lines is
not drift, while a reworded one is. A drifted surface reports the file, the line,
the exact text it says, and the canonical line it should carry.

A surface can also come back **unlocatable**: the file is there, but the locator
matches nothing in it, so the check cannot see the text at all. That is reported
in its own right and in the same breath as drift, because the reader's real
exposure is identical: a locator that has stopped matching is a surface nobody is
watching, and it would otherwise read as a clean pass. There is nothing to
propose for it, so the render offers no diff; the fix is to correct the locator or
to unregister the surface. A surface whose candidate files are all absent is
skipped rather than reported, because a file that does not exist carries no
drift.

**Autonomous rewriting is permanently out of scope.** The render proposes; the
maintainer adopts. Changing the positioning deliberately is an edit to the block,
after which the same proposal flow chases the surfaces.

Initialisation never re-interviews a repo that already has a block — it adopts it. Run
again on an adopted repo with a new title, tagline or pitch, it refuses outright
rather than overwrite the canon, and names the block to edit instead. Run again
with only a new location for the block, it writes nothing and reports where the
block is already recorded; the requested location is dropped without a line
saying so, which is a rough edge rather than a design: repointing an adopted
repo is a deliberate edit to the committed registry.

## The check

The drift check runs as the `identity-positioning` rule on every `abcd lint`,
gated on a committed registry so an un-adopted repo is skipped rather than
failed. Its acceptance corpus is
[`iss-143`](../../../work/issues/resolved/iss-143-tagline-three-variant-drift.md),
the recorded three-variant tagline drift this check exists to catch.

## References

- Plugin command: [`commands/identity.md`](../../../../commands/identity.md)
- Spec: [`spc-19`](../../specs/closed/spc-19-your-repo-says-the-same-thing-about-itself-everywhere-becaus.md)
- Intent: [`itd-102`](../../intents/shipped/itd-102-your-repo-says-the-same-thing-about-itself-everywhere-becaus.md)
- Onboarding consumer: [`15-prepare-this-repo.md`](15-prepare-this-repo.md)
- Conformance surface: [`16-lint.md`](16-lint.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd identity`

Sub-verbs: `abcd identity init`, `abcd identity render`.

Flags: none.

### `abcd identity init`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--file` | string |
| `--heading` | string |
| `--pitch` | string |
| `--tagline` | string |
| `--title` | string |

### `abcd identity render`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
