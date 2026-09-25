---
name: identity
description: "Record the identity block and propose drift corrections: Writes nothing bare, only the block and its pointer; refuses bare, naming `abcd lint identity`."
argument-hint: "[render|init]"
block: agents
---

# `/abcd:identity` repo positioning

A project's positioning fragments silently: the README strapline, the package or
plugin manifest description, and the conventions file's opening line are each
edited at different moments, until three surfaces tell three stories. This
command records the one canonical identity block and proposes the correction for
every surface that no longer says it.

## The report — what this repo says about itself

The block and every surface held to it is `/abcd:lint identity` — run `abcd lint
identity --json`. It is a status render: it exits `0` even when it reports
drift, and bare `abcd lint` is the gate. This page proposes the correction and
records the block.

## `render` — the proposed correction

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" identity render
```

prints a unified diff per drifted surface. It **writes nothing**, and no flag
makes it: adopting a proposal is the maintainer's move. Show the diff, then ask
whether to apply it. If the maintainer would rather change what the project says
than what its surfaces say, the fix is an edit to the identity block, after which
this same command chases the surfaces.

## `init` — record the block

Run this only as part of onboarding a repo, and only after the interview below.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" identity init --title "…" --tagline "…" [--pitch "…"] [--file <path>] [--heading <text>]
```

**Detect before you interview.** If the repo already carries an identity block,
`init` adopts it and leaves it byte-for-byte alone — never ask the questions
again over an existing answer, and never mint a second canon. Pass `--file` and
`--heading` when the block already lives somewhere specific. A repo whose
pointer is already committed is a no-op; passing answers to it is refused, because
repointing the canon is a deliberate edit.

### The interview

Ask once, in the maintainer's own words:

1. **Title** (required) — what the project is called, as it should read in a
   heading.
2. **Tagline** (required) — one line saying what it is. This is the line every
   surface is held to.
3. **Pitch** (optional — offer to skip) — two or three sentences a newcomer
   could read and know whether this is for them.

Then run `init` with the answers. It writes the block and the pointer
(`.abcd/positioning.json`) that records where the block lives and which surfaces
render from it — by default the README strapline, the plugin or package manifest
description, and the conventions file's opening. Further surfaces are registrable
in that same file; none are ever registered silently.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
