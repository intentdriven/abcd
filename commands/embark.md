---
name: embark
description: "List the verbs that unpack a lifeboat into a repository: Writes nothing; refuses an unknown sub-verb."
argument-hint: "probe <lifeboat> [target] | from <lifeboat> [target]"
block: people
---

# `/abcd:embark` — unpack a lifeboat into a repo

Take a packed lifeboat (produced by `/abcd:disembark`) and write its record
families back into a target repository. This is the inverse of `disembark` and the
write half of the round-trip (adr-35): ADRs, issues, intents, and specs travel
back **verbatim**, the current abcd marker block is re-injected into the target
`CLAUDE.md`, and everything else in the lifeboat informs the report but is never
written. The target defaults to the working directory when omitted.

## Probe first (read-only)

Inspect what a lifeboat would write into a target without touching either:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embark probe <lifeboat-dir> [target-dir] --json
```

The lifeboat is **untrusted input**: probe gates it, verifies its
`manifest_sha256` against the on-disk tree, and refuses a symlink or an oversize
file anywhere inside. Then it reports:

- `coverage` — the brief **blanks and their questions**, surfaced FIRST. These are
  what a lifeboat hands a product thinker: the sections nothing in the record could
  ground, each with the question a human must answer (some marked human-owned).
- `planned` — the record files that would land, each with its `family`,
  `target_path`, and `action` (`create` or, when the target already holds identical
  bytes, `unchanged`).
- `conflicts` — target paths that would block a write (a differing file, a
  non-regular target, a non-directory parent). A plan with conflicts is still a
  successful probe.
- `ignored` — lifeboat files not embarked (`report-only`, `unmapped`, `unknown`).
- `marker` — what would happen to the target `CLAUDE.md` block.
- `record_manifest_sha256` — the record-derived closure seal.

Surface the coverage blanks to the human before anything else — they are the point
of the handoff. Then summarise the plan.

## Write

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embark from <lifeboat-dir> [target-dir] --json
```

`from` runs the same planner as `probe`, then writes each `create` file into the
target through two-layer containment (an `os.Root` boundary plus independent
lexical validation), skipping `unchanged` files, and re-injects the current marker
block into the target `CLAUDE.md` — **never** foreign prose, only the canonical
block. Summarise the result: `written` / `unchanged`, the per-`families` counts,
and the `marker` action.

## What the write refuses

`from` is **conflict-safe**: if the plan carries **any** conflict, it writes
**nothing** — not a partial set — and exits non-zero with one bulk conflict report.
A conflict is per-file: a target that merely holds unrelated files is fine; a file
that already matches byte-for-byte is an idempotent skip. Relay the bulk report so
the user resolves the conflicts and re-runs. A re-run over an already-embarked
target is a clean no-op (all `unchanged`, marker `current`).

Structural faults — the directory is not an abcd lifeboat, its schema is newer than
this abcd understands, manifest verification fails, or the target is not a real
directory — exit with a single diagnostic line and write nothing.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
