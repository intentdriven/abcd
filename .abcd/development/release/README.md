# Release artefacts — the committed compatibility baseline

This directory holds the machine-generated artefacts a release is gated against.
Everything here is derived from the code and the manifests: regenerate it, never
hand-edit it.

## `surface.json` — the compatibility snapshot

`surface.json` is the structured record of abcd's public compatibility surface at
this commit:

- **every command**, by its full command path (`abcd intent plan`), with the flags
  it declares — name, shorthand, value type, requiredness, and whether the flag is
  hidden;
- **the manifest surface** — the declared key paths of `.claude-plugin/plugin.json`
  and `.claude-plugin/marketplace.json`.

Hidden commands are included. The operator-internal `hook` subtree never appears
on the documentation page, but harness wiring invokes it by name, so removing or
renaming it breaks installations exactly as a visible command would.

A manifest **entry** is one declared key path, and only its presence is recorded.
Objects flatten with dots (`author.name`); a list of objects carrying unique names
is keyed by name (`plugins[abcd].source`), so reordering it changes nothing; any
other list is one entry, so editing its members changes nothing. Values are absent
by design — a removed entry is a break, while a reworded description is not.

Nothing in the file is treated as expected or required: `plugin.json` declares no
version in the development tree, and the version the rendered release payload
carries reads as one ordinary added entry.

## What it is for

The release cut diffs the copy of this file carried by the **last release tag**
against the surface of the tree being released. A removed or renamed command, a
removed or renamed flag, a flag that becomes required, or a removed manifest entry
is a structural break, and a cut carrying one without a `breaking` record added in
that cut fails. Additions, help-text edits, and reordering are not breaks.

The copy committed here is not the comparison input — a drift test keeps it equal
to the live tree, so diffing it against that tree would compare a file with itself
and could never report a break. It is committed for two other reasons: so that the
NEXT tag carries a correct baseline for the cut after it, and so the guardrail can
check that the binary walking the surface really is this tree. The drift test
regenerates the surface from the live command tree and the live manifests on every
`go test` run and fails the build whenever the committed file disagrees, which is
what keeps both uses honest.

## Regenerating

```bash
go generate ./internal/surface/cli   # or: go run ./cmd/abcd-gen-surface
```

The generator holds no formatting logic of its own — it calls the same
`GenerateSurface` the drift test calls — so the file that is written and the file
that is checked are produced by one code path. The same run then rewrites the
generated appendix at the end of each brief surface chapter from the same walk of
the tree ([`../brief/04-surfaces/README.md` § The generated appendix](../brief/04-surfaces/README.md#the-generated-appendix)). Output is deterministic: every
collection is sorted by a stable key, the JSON key order is fixed, and nothing
reads the clock or the environment, so the same tree yields the same bytes on
every machine.

Regenerate and commit the result in the same change that alters a command, a flag,
or a manifest.

## `refusal-codes.json` — the ledger of refusal codes

`refusal-codes.json` lists every refusal code the registry in
`internal/core/refusal` holds, retired codes included, sorted, each with the
release that retired it where it is retired. Only the codes are recorded: a
code's summary, explanation and next command may be reworded in any change,
and the code may not (adr-2610301638558260). `make refusal-codes-sync` writes
it from the registry, and a drift test fails `go test` when the two disagree.

The release cut reads the copy carried by the **last release tag** and refuses
a cut in which a code that copy lists is missing from the copy at `HEAD`, or a
code it lists as retired is live again. A tag that carries no ledger passes,
so the rule binds from the first release whose ledger lists codes.

## `refusal-uncoded.txt` — the files whose refusals carry no code yet

While every refusal is being given a code (spc-2610301431108175), this list
names the files under `internal/surface/cli` that still build a refusal with
no code. The test that refuses a refusal built outside the two coded
constructors exempts only these files. It is written by hand, not generated:
the step that codes a file's last refusal removes the file from it, the list
never grows past the ceiling the test holds, and the last step deletes it.
