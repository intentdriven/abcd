# `/abcd:identity` — Repo Positioning

`/abcd:identity` holds every rendered surface of a repository to one canonical
identity block. The bare and `render` forms are **strictly read-only**; `init` is
the single write path, and it runs once, at onboarding.

It answers a different question from `/abcd:lint`: `lint` reports whether the
repo conforms to the working conventions as a whole, and runs the positioning
check as one rule among them; `identity` is where a maintainer looks at the canon
itself and at what a fix would read like.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `init` | — | shipped |
| `render` | audit | shipped |


## The identity block

The canonical home is a markdown block in the repo's own record — markdown stays
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

## The registry

`.abcd/positioning.json` records the block's location, the family severity, and
the registered surfaces. It sits beside the repo's other per-concern
configuration (`docs-lint.json`, `record-lint.json`, `rules.json`) rather than
under `.abcd/config/`, where `identity.json` already means the git
commit-author pin.

| field | meaning |
|---|---|
| `schema_version` | `1`, and **required**: `Validate` refuses any other value, the absent field included |
| `block` | `{file, heading}` — where the canonical block lives |
| `severity` | `warn` (default: highlight, never gate) or `blocker` |
| `surfaces[]` | `{id, files, kind, patterns \| field, requires, template}` |

The registry is all-or-nothing: the loader decodes with unknown fields
disallowed and validates every field before any of it is used, because each one
arrives as committed data. A registry written without `schema_version` is
refused as `schema_version must be 1, got 0`, and the positioning rule reports
an unloadable registry as a warn-tier finding rather than as drift, so the
symptom reads as a broken check rather than as a missing field. `abcd identity
init` writes the field, which is why a registry the verb laid down never hits
this.

A surface names candidate `files` (the first that exists is checked, so one entry
covers several manifest formats), a locator (`kind: "regexp"` with capture-group
patterns, or `kind: "json_field"` with a top-level key), the block fields it
`requires`, and the `template` a proposal renders from. An empty `surfaces` list
means the three defaults; a non-empty one replaces them, so nothing is ever
registered silently.

## Behaviour

```bash
abcd identity --json        # the block and every surface's verdict; exit 0
abcd identity render        # a unified diff per drifted surface; writes nothing
abcd identity init …        # record the block and the pointer to it
```

Comparison is by normalised containment: markup, dashes, line wrapping, and case
are folded away, so a tagline bolded mid-sentence or wrapped across two lines is
not drift, while a reworded one is. A drifted surface reports the file, the line,
the exact text it says, and the canonical line it should carry.

**Autonomous rewriting is permanently out of scope.** `render` proposes; the
maintainer adopts. A deliberate change of positioning is an edit to the block,
after which the same proposal flow chases the surfaces.

`init` never re-interviews a repo that already has a block — it adopts it —
and refuses to repoint a registry that is already committed.

## The check

The drift check runs as the `identity-positioning` rule on every `abcd lint`,
Where-gated on a committed registry so an un-adopted repo is skipped rather than
failed. Its acceptance corpus is [`iss-143`](../../../work/issues/resolved/iss-143-tagline-three-variant-drift.md),
the recorded three-variant tagline drift this check exists to catch.

## References

- Plugin command: [`commands/identity.md`](../../../../commands/identity.md)
- Spec: [`spc-19`](../../specs/closed/spc-19-your-repo-says-the-same-thing-about-itself-everywhere-becaus.md)
- Intent: [`itd-102`](../../intents/shipped/itd-102-your-repo-says-the-same-thing-about-itself-everywhere-becaus.md)
- Onboarding consumer: [`15-prepare-this-repo.md`](15-prepare-this-repo.md)
- Conformance surface: [`16-lint.md`](16-lint.md)
